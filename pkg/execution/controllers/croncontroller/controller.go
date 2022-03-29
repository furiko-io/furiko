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

package croncontroller

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/generated/clientset/versioned/scheme"
	executioninformers "github.com/furiko-io/furiko/pkg/generated/informers/externalversions/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
)

const (
	controllerName          = "CronController"
	waitForCacheSyncTimeout = time.Minute * 3

	// updatedConfigsBufferSize is the size of the buffered channel to process
	// JobConfig updates. To reduce duplicate work, only enqueue when the
	// JobConfig's schedule is updated.
	updatedConfigsBufferSize = 10_000
)

var (
	healthStatus uint64 = 0
)

// Controller is responsible for creating new Jobs from JobConfigs based on
// their cron schedule.
type Controller struct {
	*Context
	cronWorker     *CronWorker
	informerWorker *InformerWorker
	reconciler     *reconciler.Controller
}

// Context extends the common controllercontext.Context.
type Context struct {
	controllercontext.ContextInterface
	jobInformer       executioninformers.JobInformer
	jobconfigInformer executioninformers.JobConfigInformer
	HasSynced         []cache.InformerSynced
	queue             workqueue.RateLimitingInterface
	updatedConfigs    chan *execution.JobConfig
}

// NewContext returns a new Context.
func NewContext(context controllercontext.ContextInterface) *Context {
	c := &Context{ContextInterface: context}

	// Create workqueue.
	ratelimiter := workqueue.DefaultControllerRateLimiter()
	c.queue = workqueue.NewNamedRateLimitingQueue(ratelimiter, controllerName)

	// Bind informers.
	c.jobInformer = c.Informers().Furiko().Execution().V1alpha1().Jobs()
	c.jobconfigInformer = c.Informers().Furiko().Execution().V1alpha1().JobConfigs()
	c.HasSynced = []cache.InformerSynced{
		c.jobInformer.Informer().HasSynced,
		c.jobconfigInformer.Informer().HasSynced,
	}

	c.updatedConfigs = make(chan *execution.JobConfig, updatedConfigsBufferSize)

	return c
}

// NewRecorder returns a new Recorder for the controller.
func NewRecorder(context controllercontext.ContextInterface) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: context.Clientsets().Kubernetes().CoreV1().Events(""),
	})
	return eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})
}

func NewController(
	ctrlContext controllercontext.ContextInterface,
	concurrency *configv1.Concurrency,
) (*Controller, error) {
	ctrl := &Controller{
		Context: NewContext(ctrlContext),
	}

	ctrl.cronWorker = NewCronWorker(ctrl.Context)
	ctrl.informerWorker = NewInformerWorker(ctrl.Context)

	client := NewExecutionControl(
		(&Reconciler{}).Name(),
		ctrlContext.Clientsets().Furiko().ExecutionV1alpha1(),
		NewRecorder(ctrl.Context),
	)
	store, err := ctrlContext.Stores().ActiveJobStore()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load ActiveJobStore")
	}
	recon := NewReconciler(ctrl.Context, client, store, concurrency)
	ctrl.reconciler = reconciler.NewController(recon, ctrl.queue)

	return ctrl, nil
}

func (c *Controller) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	klog.InfoS("croncontroller: starting controller")

	// Wait for cache sync up to a timeout.
	if err := controllerutil.WaitForNamedCacheSyncWithTimeout(
		ctx,
		controllerName,
		waitForCacheSyncTimeout,
		c.HasSynced...,
	); err != nil {
		klog.ErrorS(err, "croncontroller: cache sync timeout")
		return err
	}

	atomic.StoreUint64(&healthStatus, 1)
	klog.InfoS("croncontroller: started controller")

	// Start workers
	c.reconciler.Start(ctx)
	c.cronWorker.Start(ctx.Done())

	<-ctx.Done()
	return nil
}

func (c *Controller) Shutdown(ctx context.Context) {
	klog.InfoS("croncontroller: shutting down")
	c.reconciler.Wait()
	klog.InfoS("croncontroller: stopped controller")
}

func (c *Controller) GetHealth() controllermanager.HealthStatus {
	return controllermanager.HealthStatus{
		Name:    controllerName,
		Healthy: atomic.LoadUint64(&healthStatus) == 1,
	}
}
