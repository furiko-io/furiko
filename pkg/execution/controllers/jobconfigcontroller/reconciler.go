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

package jobconfigcontroller

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/utils/execution/job"
	"github.com/furiko-io/furiko/pkg/utils/execution/jobconfig"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/logvalues"
)

type Reconciler struct {
	*Context
	concurrency *configv1.Concurrency
}

func NewReconciler(ctrlContext *Context, concurrency *configv1.Concurrency) *Reconciler {
	return &Reconciler{
		Context:     ctrlContext,
		concurrency: concurrency,
	}
}

func (w *Reconciler) Name() string {
	return fmt.Sprintf("%v.Reconciler", controllerName)
}

func (w *Reconciler) Concurrency() int {
	return controllerutil.GetConcurrencyOrDefaultCPUFactor(w.concurrency, 4)
}

func (w *Reconciler) MaxRequeues() int {
	return -1
}

func (w *Reconciler) SyncOne(ctx context.Context, namespace, name string, _ int) error {
	var err error

	trace := utiltrace.New(
		"jobconfig_sync",
		utiltrace.Field{Key: "namespace", Value: namespace},
		utiltrace.Field{Key: "name", Value: name},
	)
	defer trace.LogIfLong(500 * time.Millisecond)

	klog.V(2).InfoS("jobconfigcontroller: syncing job config",
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

	// List all Jobs for JobConfig from cache.
	rjs, err := w.listJobsForJobConfig(rjc)
	if err != nil {
		return errors.Wrapf(err, "cannot list jobs")
	}
	trace.Step("List jobs from cache done")

	// Diff against current.
	differ := newJobListDiffer(rjs)
	newJobs, inactive, removed, _ := differ.GetDiff(rjc.Status.ActiveJobs)

	// Update status.
	newRjc := rjc.DeepCopy()
	newRjc.Status.ActiveJobs = differ.GetActiveRefs()
	newRjc.Status.Active = int64(len(newRjc.Status.ActiveJobs))

	// Update list of queued jobs.
	newRjc.Status.QueuedJobs = ToJobReferences(FilterJobs(rjs, func(item execution.Job) bool {
		return job.IsQueued(&item)
	}))
	newRjc.Status.Queued = int64(len(newRjc.Status.QueuedJobs))

	// Update last schedule time.
	if lastScheduleTime := jobconfig.GetLastScheduleTime(rjs); !lastScheduleTime.IsZero() {
		newRjc.Status.LastScheduleTime = ktime.TimeMax(lastScheduleTime, newRjc.Status.LastScheduleTime)
	}

	// Compute final state.
	newRjc.Status.State = jobconfig.GetState(rjc)

	// Update JobConfig status.
	if isEqual, err := IsJobConfigStatusEqual(rjc, newRjc); err == nil && !isEqual {
		updatedRjc, err := w.Clientsets().Furiko().ExecutionV1alpha1().JobConfigs(newRjc.GetNamespace()).
			UpdateStatus(ctx, newRjc, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "cannot update job config")
		}
		rjc = updatedRjc

		klog.V(3).InfoS("jobconfigcontroller: updated job config", logvalues.
			Values("worker", w.Name(), "namespace", rjc.GetNamespace(), "name", rjc.GetName()).
			Level(4, "job_config", updatedRjc).
			Build()...,
		)

		trace.Step("Update job config done")
	}

	// Create events or log lines.
	for _, rj := range newJobs {
		klog.V(5).InfoS("jobconfigcontroller: add active job to status",
			"worker", w.Name(),
			"namespace", rjc.GetNamespace(),
			"name", rjc.GetName(),
			"job", rj.GetName(),
		)
	}

	for _, rj := range inactive {
		klog.V(5).InfoS("jobconfigcontroller: prune inactive job from status",
			"worker", w.Name(),
			"namespace", rjc.GetNamespace(),
			"name", rjc.GetName(),
			"job", rj.GetName(),
			"reason", "Finished",
		)
		w.recorder.Eventf(rjc, corev1.EventTypeNormal, "Finished", "Job %v is complete", rj.Name)
	}

	for _, name := range removed {
		klog.V(5).InfoS("jobconfigcontroller: prune inactive job from status",
			"worker", w.Name(),
			"namespace", rjc.GetNamespace(),
			"name", rjc.GetName(),
			"job", name,
			"reason", "Deleted",
		)
		w.recorder.Eventf(rjc, corev1.EventTypeNormal, "Deleted", "Job %v is removed", name)
	}

	return nil
}

func (w *Reconciler) listJobsForJobConfig(
	rjc *execution.JobConfig,
) ([]execution.Job, error) {
	labelSet := jobconfig.LabelJobsForJobConfig(rjc)
	jobs, err := w.jobInformer.Lister().Jobs(rjc.Namespace).List(labels.SelectorFromSet(labelSet))
	if err != nil {
		return nil, errors.Wrapf(err, "could not list jobs")
	}

	rjobs := make([]execution.Job, 0, len(jobs))
	for _, rj := range jobs {
		rjobs = append(rjobs, *rj)
	}

	return rjobs, nil
}
