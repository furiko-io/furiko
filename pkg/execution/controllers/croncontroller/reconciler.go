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
	"fmt"
	"time"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	utiltrace "k8s.io/utils/trace"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/utils/execution/jobconfig"
)

// Reconciler creates Jobs that are queued to be started from the workqueue,
// with a schedule time and JobConfig, unless forbidden by the
// ConcurrencyPolicy. It may be possible for multiple workers to process the
// same JobConfig concurrently.
type Reconciler struct {
	*Context
	concurrency *configv1alpha1.Concurrency
	client      ExecutionControlInterface
	recorder    Recorder
	store       controllercontext.ActiveJobStore
}

func NewReconciler(
	ctrlContext *Context,
	client ExecutionControlInterface,
	recorder Recorder,
	store controllercontext.ActiveJobStore,
	concurrency *configv1alpha1.Concurrency,
) *Reconciler {
	return &Reconciler{
		Context:     ctrlContext,
		concurrency: concurrency,
		client:      client,
		recorder:    recorder,
		store:       store,
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
	configName, ts, err := SplitJobConfigKeyName(name)
	if err != nil {
		return errors.Wrapf(err, "could not split key")
	}
	return w.processCronForConfig(ctx, namespace, configName, ts)
}

func (w *Reconciler) processCronForConfig(ctx context.Context, namespace, name string, scheduleTime time.Time) error {
	trace := utiltrace.New(
		"cron_reconcile",
		utiltrace.Field{Key: "namespace", Value: namespace},
		utiltrace.Field{Key: "name", Value: name},
		utiltrace.Field{Key: "schedule_time", Value: scheduleTime},
	)
	defer trace.LogIfLong(500 * time.Millisecond)

	cfg, err := w.Configs().JobConfigs()
	if err != nil {
		return errors.Wrapf(err, "cannot get jobconfig configuration")
	}

	// Get JobConfig from cache
	jobConfig, err := w.jobconfigInformer.Lister().JobConfigs(namespace).Get(name)

	// If JobConfig is not found, it is an invalid one (or it was deleted).
	if kerrors.IsNotFound(err) {
		return nil
	}
	trace.Step("Lookup jobconfig in cache done")

	// Count active jobs for the JobConfig.
	activeJobCount := w.store.CountActiveJobsForConfig(jobConfig)
	trace.Step("Count active jobs for config done")

	// Check concurrency policy.
	concurrencyPolicy := jobConfig.Spec.Concurrency.Policy

	// Handle Forbid concurrency policy.
	if concurrencyPolicy == execution.ConcurrencyPolicyForbid && activeJobCount > 0 {
		w.recorder.SkippedJobSchedule(ctx, jobConfig, scheduleTime,
			"Skipped creating job due to concurrency policy Forbid")
		return nil
	}

	// Cannot enqueue beyond max queue length.
	// TODO(irvinlim): We use the status here, which may not be fully up-to-date.
	if max := cfg.MaxEnqueuedJobs; max != nil && jobConfig.Status.Queued >= *max {
		w.recorder.SkippedJobSchedule(ctx, jobConfig, scheduleTime,
			fmt.Sprintf("Skipped creating job, cannot exceed maximum queue length of %v", *max))
		return nil
	}

	// Initialise a new Job object.
	newJob, err := jobconfig.NewJobFromJobConfig(jobConfig, execution.JobTypeScheduled, scheduleTime)
	if err != nil {
		return errors.Wrapf(err, "could not create Job")
	}
	trace.Step("Init new job done")

	// Set start policy.
	newJob.Spec.StartPolicy = &execution.StartPolicySpec{
		ConcurrencyPolicy: concurrencyPolicy,
	}

	// Look up existing job in cache as a quick way to bail.
	// Safe to use cache since server will tell us if we are creating a duplicate Job.
	_, err = w.jobInformer.Lister().Jobs(newJob.GetNamespace()).Get(newJob.GetName())
	if err != nil && !kerrors.IsNotFound(err) {
		return errors.Wrapf(err, "could not get job")
	}

	// If existing Job does not exist, create the resource.
	if kerrors.IsNotFound(err) {
		if err := w.client.CreateJob(ctx, jobConfig, newJob); err != nil {
			return errors.Wrapf(err, "could not create new job")
		}
		trace.Step("Create job done")
	}

	return nil
}
