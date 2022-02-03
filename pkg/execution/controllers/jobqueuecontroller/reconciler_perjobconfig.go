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
	"sort"
	"time"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/utils/execution/job"
	"github.com/furiko-io/furiko/pkg/utils/execution/jobconfig"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	timeutil "github.com/furiko-io/furiko/pkg/utils/time"
)

// PerConfigReconciler reconciles Jobs belonging to a JobConfig. It is
// guaranteed that no two routines will process Jobs belonging to the same
// JobConfig.
type PerConfigReconciler struct {
	*Context
	concurrency *configv1.Concurrency
	client      JobControlInterface
}

func NewPerConfigReconciler(ctrlContext *Context, concurrency *configv1.Concurrency) *PerConfigReconciler {
	reconciler := &PerConfigReconciler{
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

func (w *PerConfigReconciler) Name() string {
	return fmt.Sprintf("%v.Reconciler", controllerName)
}

func (w *PerConfigReconciler) Concurrency() int {
	return controllerutil.GetConcurrencyOrDefaultCPUFactor(w.concurrency, 4)
}

func (w *PerConfigReconciler) MaxRequeues() int {
	return -1
}

func (w *PerConfigReconciler) SyncOne(ctx context.Context, namespace, name string, _ int) error {
	var err error

	trace := utiltrace.New(
		"jobqueue_perjobconfig_sync",
		utiltrace.Field{Key: "namespace", Value: namespace},
		utiltrace.Field{Key: "name", Value: name},
	)
	defer trace.LogIfLong(500 * time.Millisecond)

	klog.V(2).InfoS("jobqueuecontroller: syncing job config",
		"worker", w.Name(),
		"namespace", namespace,
		"name", name,
	)

	rjc, err := w.jobconfigInformer.Lister().JobConfigs(namespace).Get(name)
	if kerrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "cannot get job config")
	}
	trace.Step("Lookup job config from cache done")

	// List queued Jobs for JobConfig in order of creation time.
	rjs, err := w.listQueuedJobsForJobConfig(rjc)
	if err != nil {
		return errors.Wrapf(err, "cannot list jobs")
	}
	trace.Step("List unstarted jobs from cache done")

	if len(rjs) == 0 {
		return nil
	}

	// Get a snapshot of the current number of active jobs.
	store, err := w.Stores().ActiveJobStore()
	if err != nil {
		return errors.Wrapf(err, "cannot get activejobstore")
	}
	activeCount := store.CountActiveJobsForConfig(rjc)

	// Start all Jobs that we can start in order of oldest to newest. Note that we
	// cannot continue on error, we have to retry the whole routine in order to
	// avoid violating the start order.
	for _, rj := range rjs {
		ok, err := w.canStartJob(ctx, rjc, rj, activeCount)
		if err != nil {
			return errors.Wrapf(err, "cannot check if job can start")
		}
		if !ok {
			continue
		}
		if err := w.startJob(ctx, rj, store, activeCount); err != nil {
			return errors.Wrapf(err, "cannot start job")
		}

		// Update activeCount here, since it would already have been updated in the previous step.
		activeCount = store.CountActiveJobsForConfig(rjc)
	}

	return nil
}

func (w *PerConfigReconciler) listQueuedJobsForJobConfig(
	rjc *execution.JobConfig,
) ([]*execution.Job, error) {
	labelSet := jobconfig.LabelJobsForJobConfig(rjc)
	jobs, err := w.jobInformer.Lister().Jobs(rjc.Namespace).List(labels.SelectorFromSet(labelSet))
	if err != nil {
		return nil, errors.Wrapf(err, "could not list jobs")
	}

	rjobs := make([]*execution.Job, 0, len(jobs))
	for _, rj := range jobs {
		if job.IsQueued(rj) {
			rjobs = append(rjobs, rj)
		}
	}

	// Sort by creation timestamp.
	sort.Slice(rjobs, func(i, j int) bool {
		return rjobs[i].CreationTimestamp.Before(&rjobs[j].CreationTimestamp)
	})

	return rjobs, nil
}

func (w *PerConfigReconciler) canStartJob(
	ctx context.Context,
	rjc *execution.JobConfig,
	rj *execution.Job,
	activeCount int64,
) (bool, error) {
	if spec := rj.Spec.StartPolicy; spec != nil {
		// Cannot start yet.
		if ktime.IsTimeSetAndLater(spec.StartAfter) {
			w.enqueueAfter(rjc, "job_start_after", time.Until(spec.StartAfter.Time))
			return false, nil
		}

		// There are concurrent jobs and we should immediately reject the job.
		if spec.ConcurrencyPolicy == execution.ConcurrencyPolicyForbid && activeCount > 0 {
			msg := fmt.Sprintf("Cannot start new Job, %v has %v active Jobs but concurrency policy is %v",
				rjc.Name, activeCount, spec.ConcurrencyPolicy)
			if err := w.client.RejectJob(ctx, rj, msg); err != nil {
				return false, errors.Wrapf(err, "failed to reject job")
			}
			klog.InfoS("jobqueuecontroller: job rejected due to concurrency policy",
				"worker", w.Name(),
				"namespace", rj.GetNamespace(),
				"name", rj.GetName(),
				"active_count", activeCount,
				"concurrency_policy", spec.ConcurrencyPolicy,
			)

			return false, nil
		}

		// There are concurrent jobs and we should wait.
		if spec.ConcurrencyPolicy == execution.ConcurrencyPolicyEnqueue && activeCount > 0 {
			return false, nil
		}
	}

	return true, nil
}

func (w *PerConfigReconciler) startJob(
	ctx context.Context,
	rj *execution.Job,
	store controllercontext.ActiveJobStore,
	oldCount int64,
) (err error) {
	// Atomically update the count, otherwise return an error to retry creation via workqueue.
	if !store.CheckAndAdd(rj, oldCount) {
		return fmt.Errorf("concurrent add, try again")
	}
	defer func() {
		if err != nil {
			store.Delete(rj)
		}
	}()

	return w.client.StartJob(ctx, rj)
}

// enqueueAfter will defer a sync after the specified duration, and logs the purpose of deferring
// the sync for debugging purposes.
// We enforce a lower bound of 1 second to the next sync, to slow down unwanted bursts of syncs.
func (w *PerConfigReconciler) enqueueAfter(rjc *execution.JobConfig, purpose string, duration time.Duration) {
	duration = timeutil.DurationMax(time.Second, duration)
	if key, err := cache.MetaNamespaceKeyFunc(rjc); err == nil {
		w.jobConfigQueue.AddAfter(key, duration)
		klog.V(2).InfoS("jobqueuecontroller: worker enqueue sync",
			"worker", w.Name(),
			"namespace", rjc.GetNamespace(),
			"name", rjc.GetName(),
			"purpose", purpose,
			"after", duration.String(),
		)
	}
}
