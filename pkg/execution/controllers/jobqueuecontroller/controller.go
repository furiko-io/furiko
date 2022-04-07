/*
 * Copyright 2022 The Furiko Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jobqueuecontroller

import (
	"context"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	"github.com/furiko-io/furiko/pkg/generated/clientset/versioned/scheme"
	executioninformers "github.com/furiko-io/furiko/pkg/generated/informers/externalversions/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
)

const (
	controllerName          = "JobQueueController"
	waitForCacheSyncTimeout = time.Minute * 3
)

var (
	healthStatus uint64
)

// Controller is responsible for processing all Jobs to determine if they can be
// started, and if so, starts them in a deterministic order and safe from race
// conditions.
type Controller struct {
	*Context
	informerWorker        *InformerWorker
	perConfigReconciler   *reconciler.Controller
	independentReconciler *reconciler.Controller
}

// Context extends the common controllercontext.Context.
type Context struct {
	controllercontext.ContextInterface
	jobInformer       executioninformers.JobInformer
	jobconfigInformer executioninformers.JobConfigInformer
	hasSynced         []cache.InformerSynced
	jobConfigQueue    workqueue.RateLimitingInterface
	independentQueue  workqueue.RateLimitingInterface
	recorder          record.EventRecorder
}

func NewContext(context controllercontext.ContextInterface) *Context {
	c := &Context{ContextInterface: context}

	// Create recorder.
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: c.Clientsets().Kubernetes().CoreV1().Events(""),
	})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})

	// Bind informers.
	c.jobInformer = c.Informers().Furiko().Execution().V1alpha1().Jobs()
	c.jobconfigInformer = c.Informers().Furiko().Execution().V1alpha1().JobConfigs()
	c.hasSynced = []cache.InformerSynced{
		c.jobInformer.Informer().HasSynced,
		c.jobconfigInformer.Informer().HasSynced,
	}

	return c
}

func NewController(
	ctrlContext controllercontext.ContextInterface,
	concurrency *configv1.Concurrency,
) (*Controller, error) {
	ctrl := &Controller{
		Context: NewContext(ctrlContext),
	}

	// Create multiple reconcilers. For Jobs that are not owned by a JobConfig, we
	// will simply start them as soon as they are created. Otherwise, we will
	// sequentially process Jobs on a per-JobConfig basis in each Reconciler
	// goroutine.
	ratelimiter := workqueue.DefaultControllerRateLimiter()
	perConfigReconciler := NewPerConfigReconciler(ctrl.Context, concurrency)
	ctrl.Context.jobConfigQueue = workqueue.NewNamedRateLimitingQueue(ratelimiter, perConfigReconciler.Name())
	independentReconciler := NewIndependentReconciler(ctrl.Context, concurrency)
	ctrl.Context.independentQueue = workqueue.NewNamedRateLimitingQueue(ratelimiter, independentReconciler.Name())

	ctrl.informerWorker = NewInformerWorker(ctrl.Context)
	ctrl.perConfigReconciler = reconciler.NewController(perConfigReconciler, ctrl.jobConfigQueue)
	ctrl.independentReconciler = reconciler.NewController(independentReconciler, ctrl.independentQueue)

	return ctrl, nil
}

func (c *Controller) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	defer c.jobConfigQueue.ShutDown()
	defer c.independentQueue.ShutDown()
	klog.InfoS("jobqueuecontroller: starting controller")

	// Wait for cache sync up to a timeout.
	if err := controllerutil.WaitForNamedCacheSyncWithTimeout(
		ctx,
		controllerName,
		waitForCacheSyncTimeout,
		c.hasSynced...,
	); err != nil {
		klog.ErrorS(err, "jobqueuecontroller: cache sync timeout")
		return err
	}

	atomic.StoreUint64(&healthStatus, 1)
	klog.InfoS("jobqueuecontroller: started controller")

	// Start workers
	c.perConfigReconciler.Start(ctx)
	c.independentReconciler.Start(ctx)

	<-ctx.Done()
	return nil
}

func (c *Controller) Shutdown(ctx context.Context) {
	klog.InfoS("jobqueuecontroller: shutting down")
	c.perConfigReconciler.Wait()
	klog.InfoS("jobqueuecontroller: stopped controller")
}

func (c *Controller) GetHealth() controllermanager.HealthStatus {
	return controllermanager.HealthStatus{
		Name:    controllerName,
		Healthy: atomic.LoadUint64(&healthStatus) == 1,
	}
}
