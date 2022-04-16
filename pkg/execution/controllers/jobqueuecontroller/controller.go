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

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/generated/clientset/versioned/scheme"
	executioninformers "github.com/furiko-io/furiko/pkg/generated/informers/externalversions/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
)

// Controller is responsible for processing all Jobs to determine if they can be
// started, and if so, starts them in a deterministic order and safe from race
// conditions.
type Controller struct {
	*Context
	ctx                   context.Context
	terminate             context.CancelFunc
	healthStatus          uint64
	informerWorker        *InformerWorker
	perConfigReconciler   *reconciler.Controller
	independentReconciler *reconciler.Controller
}

// Context extends the common controllercontext.Context.
type Context struct {
	controllercontext.Context
	jobInformer       executioninformers.JobInformer
	jobconfigInformer executioninformers.JobConfigInformer
	HasSynced         []cache.InformerSynced
	jobConfigQueue    workqueue.RateLimitingInterface
	independentQueue  workqueue.RateLimitingInterface
	recorder          record.EventRecorder
}

// NewContext returns a new Context.
func NewContext(context controllercontext.Context) *Context {
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

	// Set recorder
	c.recorder = recorder

	// Bind informers.
	c.jobInformer = c.Informers().Furiko().Execution().V1alpha1().Jobs()
	c.jobconfigInformer = c.Informers().Furiko().Execution().V1alpha1().JobConfigs()
	c.HasSynced = []cache.InformerSynced{
		c.jobInformer.Informer().HasSynced,
		c.jobconfigInformer.Informer().HasSynced,
	}

	// Create workqueues.
	c.jobConfigQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(),
		(&PerConfigReconciler{}).Name())
	c.independentQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(),
		(&IndependentReconciler{}).Name())

	return c
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

	// Create multiple reconcilers. For Jobs that are not owned by a JobConfig, we
	// will simply start them as soon as they are created. Otherwise, we will
	// sequentially process Jobs on a per-JobConfig basis in each Reconciler
	// goroutine.
	perConfigReconciler := NewPerConfigReconciler(ctrl.Context, concurrency)
	independentReconciler := NewIndependentReconciler(ctrl.Context, concurrency)

	ctrl.informerWorker = NewInformerWorker(ctrl.Context)
	ctrl.perConfigReconciler = reconciler.NewController(perConfigReconciler, ctrl.jobConfigQueue)
	ctrl.independentReconciler = reconciler.NewController(independentReconciler, ctrl.independentQueue)

	return ctrl, nil
}

func (c *Controller) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	klog.InfoS("jobqueuecontroller: starting controller")

	if ok := cache.WaitForNamedCacheSync(controllerName, ctx.Done(), c.HasSynced...); !ok {
		klog.Error("jobqueuecontroller: cache sync timeout")
		return controllerutil.ErrWaitForCacheSyncTimeout
	}

	c.perConfigReconciler.Start(c.ctx)
	c.independentReconciler.Start(c.ctx)

	atomic.StoreUint64(&c.healthStatus, 1)
	klog.InfoS("jobqueuecontroller: started controller")

	return nil
}

func (c *Controller) Shutdown(ctx context.Context) {
	klog.InfoS("jobqueuecontroller: shutting down")
	c.terminate()
	c.jobConfigQueue.ShutDown()
	c.independentQueue.ShutDown()
	c.perConfigReconciler.Wait()
	klog.InfoS("jobqueuecontroller: stopped controller")
}

func (c *Controller) GetHealth() controllermanager.HealthStatus {
	return controllermanager.HealthStatus{
		Name:    controllerName,
		Healthy: atomic.LoadUint64(&c.healthStatus) == 1,
	}
}
