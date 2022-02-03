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
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/utils/execution/jobconfig"
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
	configName, ts, err := SplitJobConfigKeyName(name)
	if err != nil {
		return errors.Wrapf(err, "could not split key")
	}
	return w.processCronForConfig(ctx, namespace, configName, ts)
}

// processCronForConfig creates a single Job in response to cron scheduling. We
// will create a new Job unless forbidden by the ConcurrencyPolicy. It may be
// possible for multiple worker instances to process the same JobConfig
// concurrently.
func (w *Reconciler) processCronForConfig(ctx context.Context, namespace, name string, scheduleTime time.Time) error {
	trace := utiltrace.New(
		"cron_reconcile",
		utiltrace.Field{Key: "namespace", Value: namespace},
		utiltrace.Field{Key: "name", Value: name},
		utiltrace.Field{Key: "schedule_time", Value: scheduleTime},
	)
	defer trace.LogIfLong(500 * time.Millisecond)

	// Get JobConfig from cache
	jobConfig, err := w.jobconfigInformer.Lister().JobConfigs(namespace).Get(name)

	// If JobConfig is not found, it is an invalid one (or it was deleted).
	if kerrors.IsNotFound(err) {
		return nil
	}
	trace.Step("Lookup jobconfig in cache done")

	// Count active jobs for the JobConfig.
	store, err := w.Stores().ActiveJobStore()
	if err != nil {
		return errors.Wrapf(err, "cannot get activejobstore")
	}
	activeJobCount := store.CountActiveJobsForConfig(jobConfig)
	trace.Step("Count active jobs for config done")

	// Check concurrency policy.
	concurrencyPolicy := jobConfig.Spec.Concurrency.Policy

	// Skip if not allowed.
	if concurrencyPolicy == execution.ConcurrencyPolicyForbid && activeJobCount > 0 {
		klog.V(3).InfoS("croncontroller: skip create job due to concurrency policy",
			"worker", w.Name(),
			"namespace", jobConfig.GetNamespace(),
			"name", jobConfig.GetName(),
			"schedule_time", scheduleTime,
			"concurrency_policy", concurrencyPolicy,
			"active_count", activeJobCount,
		)
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
		if err := w.createNewJob(ctx, jobConfig, newJob); err != nil {
			return errors.Wrapf(err, "could not create new job")
		}
		trace.Step("Create job done")
	}

	return nil
}

func (w *Reconciler) createNewJob(
	ctx context.Context,
	rjc *execution.JobConfig,
	rj *execution.Job,
) (err error) {
	createdRj, err := w.Clientsets().Furiko().ExecutionV1alpha1().Jobs(rj.GetNamespace()).
		Create(ctx, rj, metav1.CreateOptions{})

	// If we fail to create the Job due to an invalid error (e.g. from webhook), do
	// not retry and instead store as an event.
	if kerrors.IsInvalid(err) {
		// Get detailed error information if possible.
		errMessage := err.Error()
		if err, ok := err.(kerrors.APIStatus); ok {
			if details := err.Status().Details; details != nil && len(details.Causes) > 0 {
				causes := make([]string, 0, len(details.Causes))
				for _, cause := range details.Causes {
					causes = append(causes, fmt.Sprintf("%s: %s", cause.Field, cause.Message))
				}
				errMessage = strings.Join(causes, ", ")
			}
		}

		klog.ErrorS(err, "croncontroller: failed to create job", logvalues.
			Values("worker", w.Name(), "namespace", rj.GetNamespace(), "name", rj.GetName()).
			Level(4, "job", rj).
			Build()...,
		)
		w.recorder.Eventf(rjc, corev1.EventTypeWarning, "CronScheduleFailed",
			"Failed to create Job %v from cron schedule: %v", rj.GetName(), errMessage)
		return nil
	}

	if err != nil {
		return errors.Wrapf(err, "cannot create job")
	}

	klog.InfoS("croncontroller: created job", logvalues.
		Values("worker", w.Name(), "namespace", createdRj.GetNamespace(), "name", createdRj.GetName()).
		Level(4, "job", createdRj).
		Build()...,
	)

	w.recorder.Eventf(rjc, corev1.EventTypeNormal, "CronSchedule",
		"Created Job %v from cron schedule", createdRj.GetName())

	return nil
}
