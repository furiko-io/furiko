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
	"fmt"
	"time"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	timeutil "github.com/furiko-io/furiko/pkg/utils/time"
)

// IndependentReconciler reconciles independent Jobs that do not have any JobConfig.
type IndependentReconciler struct {
	*Context
	concurrency *configv1alpha1.Concurrency
	client      JobControlInterface
}

func NewIndependentReconciler(ctrlContext *Context, concurrency *configv1alpha1.Concurrency) *IndependentReconciler {
	reconciler := &IndependentReconciler{
		Context:     ctrlContext,
		concurrency: concurrency,
	}
	reconciler.client = NewJobControl(
		ctrlContext.Clientsets().Furiko().ExecutionV1alpha1(),
		ctrlContext.recorder,
		reconciler.Name(),
	)
	return reconciler
}

func (r *IndependentReconciler) Name() string {
	return fmt.Sprintf("%v.Reconciler", controllerName)
}

func (r *IndependentReconciler) Concurrency() int {
	return controllerutil.GetConcurrencyOrDefaultCPUFactor(r.concurrency, 4)
}

func (r *IndependentReconciler) MaxRequeues() int {
	return -1
}

func (r *IndependentReconciler) SyncOne(ctx context.Context, namespace, name string, _ int) error {
	var err error

	trace := utiltrace.New(
		"jobqueue_independent_sync",
		utiltrace.Field{Key: "namespace", Value: namespace},
		utiltrace.Field{Key: "name", Value: name},
	)
	defer trace.LogIfLong(500 * time.Millisecond)

	klog.V(2).InfoS("jobqueuecontroller: syncing job",
		"worker", r.Name(),
		"namespace", namespace,
		"name", name,
	)

	rj, err := r.jobInformer.Lister().Jobs(namespace).Get(name)
	if kerrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "cannot get job")
	}
	trace.Step("Lookup job from cache done")

	if !job.IsQueued(rj) {
		return nil
	}

	if spec := rj.Spec.StartPolicy; spec != nil {
		if ktime.IsTimeSetAndLater(spec.StartAfter) {
			r.enqueueAfter(rj, "job_start_after", time.Until(spec.StartAfter.Time))
			return nil
		}
	}

	if err := r.client.StartJob(ctx, rj); err != nil {
		return errors.Wrapf(err, "cannot start job")
	}
	trace.Step("Start job done")

	return nil
}

// enqueueAfter will defer a sync after the specified duration, and logs the purpose of deferring
// the sync for debugging purposes.
// We enforce a lower bound of 1 second to the next sync, to slow down unwanted bursts of syncs.
func (r *IndependentReconciler) enqueueAfter(rj *execution.Job, purpose string, duration time.Duration) {
	duration = timeutil.DurationMax(time.Second, duration)
	if key, err := cache.MetaNamespaceKeyFunc(rj); err == nil {
		r.independentQueue.AddAfter(key, duration)
		klog.V(2).InfoS("jobqueuecontroller: worker enqueue sync",
			"worker", r.Name(),
			"namespace", rj.GetNamespace(),
			"name", rj.GetName(),
			"purpose", purpose,
			"after", duration.String(),
		)
	}
}
