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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/generated/clientset/versioned/scheme"
	executioninformers "github.com/furiko-io/furiko/pkg/generated/informers/externalversions/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
)

const (
	// updatedConfigsBufferSize is the size of the buffered channel to process
	// JobConfig updates. To reduce duplicate work, only enqueue when the
	// JobConfig's schedule is updated.
	updatedConfigsBufferSize = 10_000
)

// Controller is responsible for creating new Jobs from JobConfigs based on
// their cron schedule.
type Controller struct {
	*Context
	ctx            context.Context
	terminate      context.CancelFunc
	healthStatus   uint64
	cronWorker     *CronWorker
	informerWorker *InformerWorker
	reconciler     *reconciler.Controller
}

// Context extends the common controllercontext.Context.
type Context struct {
	controllercontext.Context
	jobInformer       executioninformers.JobInformer
	jobconfigInformer executioninformers.JobConfigInformer
	HasSynced         []cache.InformerSynced
	Queue             workqueue.RateLimitingInterface
	UpdatedConfigs    chan *execution.JobConfig
}

// NewContext returns a new Context.
func NewContext(context controllercontext.Context) *Context {
	c := &Context{Context: context}

	// Create workqueue.
	ratelimiter := workqueue.DefaultControllerRateLimiter()
	c.Queue = workqueue.NewNamedRateLimitingQueue(ratelimiter, controllerName)

	// Bind informers.
	c.jobInformer = c.Informers().Furiko().Execution().V1alpha1().Jobs()
	c.jobconfigInformer = c.Informers().Furiko().Execution().V1alpha1().JobConfigs()
	c.HasSynced = []cache.InformerSynced{
		c.jobInformer.Informer().HasSynced,
		c.jobconfigInformer.Informer().HasSynced,
	}

	c.UpdatedConfigs = make(chan *execution.JobConfig, updatedConfigsBufferSize)

	return c
}

// NewRecorder returns a new Recorder for the controller.
func NewRecorder(context controllercontext.Context) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: context.Clientsets().Kubernetes().CoreV1().Events(""),
	})
	return eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})
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
	ctrl.reconciler = reconciler.NewController(recon, ctrl.Queue)

	return ctrl, nil
}

func (c *Controller) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	klog.InfoS("croncontroller: starting controller")

	if ok := cache.WaitForNamedCacheSync(controllerName, ctx.Done(), c.HasSynced...); !ok {
		klog.Error("croncontroller: cache sync timeout")
		return controllerutil.ErrWaitForCacheSyncTimeout
	}

	c.reconciler.Start(c.ctx)
	c.cronWorker.Start(c.ctx)

	atomic.StoreUint64(&c.healthStatus, 1)
	klog.InfoS("croncontroller: started controller")

	return nil
}

func (c *Controller) Shutdown(ctx context.Context) {
	klog.InfoS("croncontroller: shutting down")
	c.terminate()
	c.Queue.ShutDown()
	c.reconciler.Wait()
	klog.InfoS("croncontroller: stopped controller")
}

func (c *Controller) GetHealth() controllermanager.HealthStatus {
	return controllermanager.HealthStatus{
		Name:    controllerName,
		Healthy: atomic.LoadUint64(&c.healthStatus) == 1,
	}
}
