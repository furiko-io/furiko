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

package jobcontroller

import (
	"context"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor"
	"github.com/furiko-io/furiko/pkg/generated/clientset/versioned/scheme"
	executioninformers "github.com/furiko-io/furiko/pkg/generated/informers/externalversions/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
)

const (
	controllerName          = "JobController"
	waitForCacheSyncTimeout = time.Minute * 3
)

var (
	healthStatus uint64 = 0
)

// Controller is responsible for reconciling Jobs with their downstream task
// executor states.
type Controller struct {
	*Context
	informerWorker *InformerWorker
	reconciler     *reconciler.Controller
}

// Context extends the common controllercontext.Context.
type Context struct {
	controllercontext.ContextInterface
	podInformer coreinformers.PodInformer
	jobInformer executioninformers.JobInformer
	hasSynced   []cache.InformerSynced
	queue       workqueue.RateLimitingInterface
	recorder    record.EventRecorder
	tasks       *taskexecutor.Manager
}

func NewContext(context controllercontext.ContextInterface) *Context {
	c := &Context{ContextInterface: context}

	// Create recorder.
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: c.Clientsets().Kubernetes().CoreV1().Events(""),
	})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})

	// Create workqueue.
	ratelimiter := workqueue.DefaultControllerRateLimiter()
	c.queue = workqueue.NewNamedRateLimitingQueue(ratelimiter, controllerName)

	// Bind informers.
	c.podInformer = c.Informers().Kubernetes().Core().V1().Pods()
	c.jobInformer = c.Informers().Furiko().Execution().V1alpha1().Jobs()
	c.hasSynced = []cache.InformerSynced{
		c.podInformer.Informer().HasSynced,
		c.jobInformer.Informer().HasSynced,
	}

	// Add task manager
	c.tasks = taskexecutor.NewManager(context.Clientsets(), context.Informers())

	return c
}

func NewController(
	ctrlContext controllercontext.ContextInterface,
	concurrency *configv1.Concurrency,
) (*Controller, error) {
	ctrl := &Controller{
		Context: NewContext(ctrlContext),
	}

	ctrl.informerWorker = NewInformerWorker(ctrl.Context)
	ctrl.reconciler = reconciler.NewController(NewReconciler(ctrl.Context, concurrency), ctrl.queue)

	return ctrl, nil
}

func (c *Controller) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	klog.InfoS("jobcontroller: starting controller")

	// Wait for cache sync up to a timeout.
	if err := controllerutil.WaitForNamedCacheSyncWithTimeout(
		ctx,
		controllerName,
		waitForCacheSyncTimeout,
		c.hasSynced...,
	); err != nil {
		klog.ErrorS(err, "jobcontroller: cache sync timeout")
		return err
	}

	atomic.StoreUint64(&healthStatus, 1)
	klog.InfoS("jobcontroller: started controller")

	// Start workers
	c.reconciler.Start(ctx)

	<-ctx.Done()
	return nil
}

func (c *Controller) Shutdown(ctx context.Context) {
	klog.InfoS("jobcontroller: shutting down")
	c.reconciler.Wait()
	klog.InfoS("jobcontroller: stopped controller")
}

func (c *Controller) GetHealth() controllermanager.HealthStatus {
	return controllermanager.HealthStatus{
		Name:    controllerName,
		Healthy: atomic.LoadUint64(&healthStatus) == 1,
	}
}
