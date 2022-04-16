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

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/generated/clientset/versioned/scheme"
	executioninformers "github.com/furiko-io/furiko/pkg/generated/informers/externalversions/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
)

// Controller is responsible for reconciling Jobs with their downstream task
// executor states.
type Controller struct {
	*Context
	ctx            context.Context
	terminate      context.CancelFunc
	healthStatus   uint64
	informerWorker *InformerWorker
	reconciler     *reconciler.Controller
}

// Context extends the common controllercontext.Context.
type Context struct {
	controllercontext.Context
	podInformer coreinformers.PodInformer
	jobInformer executioninformers.JobInformer
	hasSynced   []cache.InformerSynced
	queue       workqueue.RateLimitingInterface
	recorder    record.EventRecorder
	tasks       tasks.ExecutorFactory
}

// NewContext returns a new Context.
func NewContext(context controllercontext.Context) *Context {
	// Create recorder.
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: context.Clientsets().Kubernetes().CoreV1().Events(""),
	})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})

	return NewContextWithRecorder(context, recorder)
}

// NewContextWithRecorder returns a new Context with a custom EventRecorder.
func NewContextWithRecorder(context controllercontext.Context, recorder record.EventRecorder) *Context {
	c := &Context{Context: context}

	// Set recorder.
	c.recorder = recorder

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

	// Set task manager.
	c.tasks = taskexecutor.NewManager(context.Clientsets(), context.Informers())

	return c
}

func (c *Context) GetHasSynced() []cache.InformerSynced {
	return c.hasSynced
}

func NewController(
	ctrlContext controllercontext.Context,
	concurrency *configv1alpha1.Concurrency,
) (*Controller, error) {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := &Controller{
		Context:   NewContext(ctrlContext),
		ctx:       ctx,
		terminate: cancel,
	}

	ctrl.informerWorker = NewInformerWorker(ctrl.Context)
	ctrl.reconciler = reconciler.NewController(NewReconciler(ctrl.Context, concurrency), ctrl.queue)

	return ctrl, nil
}

func (c *Controller) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	klog.InfoS("jobcontroller: starting controller")

	// Wait for cache sync up to a timeout.
	if ok := cache.WaitForNamedCacheSync(controllerName, ctx.Done(), c.hasSynced...); !ok {
		klog.Error("jobcontroller: cache sync timeout")
		return controllerutil.ErrWaitForCacheSyncTimeout
	}

	c.reconciler.Start(c.ctx)

	atomic.StoreUint64(&c.healthStatus, 1)
	klog.InfoS("jobcontroller: started controller")

	return nil
}

func (c *Controller) Shutdown(ctx context.Context) {
	klog.InfoS("jobcontroller: shutting down")
	c.terminate()
	c.queue.ShutDown()
	c.reconciler.Wait()
	klog.InfoS("jobcontroller: stopped controller")
}

func (c *Controller) GetHealth() controllermanager.HealthStatus {
	return controllermanager.HealthStatus{
		Name:    controllerName,
		Healthy: atomic.LoadUint64(&c.healthStatus) == 1,
	}
}
