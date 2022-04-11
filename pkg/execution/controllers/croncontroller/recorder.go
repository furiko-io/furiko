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
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/generated/clientset/versioned/scheme"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/utils/logvalues"
)

// Recorder knows how to record events for the CronController.
type Recorder interface {
	// CreatedJob creates an event when a JobConfig created a new Job.
	CreatedJob(ctx context.Context, jobConfig *execution.JobConfig, job *execution.Job)

	// CreateJobFailed creates an event when a JobConfig failed to create a new Job.
	CreateJobFailed(ctx context.Context, jobConfig *execution.JobConfig, job *execution.Job, message string)

	// SkippedJobSchedule creates an event whenever a JobConfig's schedule was skipped for whatever reason.
	SkippedJobSchedule(ctx context.Context, jobConfig *execution.JobConfig, scheduleTime time.Time, message string)
}

type defaultRecorder struct {
	name     string
	recorder record.EventRecorder
}

func newDefaultRecorder(ctrlContext *Context, name string) *defaultRecorder {
	return &defaultRecorder{name: name, recorder: newEventRecorder(ctrlContext)}
}

var _ Recorder = (*defaultRecorder)(nil)

func (r *defaultRecorder) CreatedJob(_ context.Context, jobConfig *execution.JobConfig, job *execution.Job) {
	klog.InfoS("croncontroller: created job", logvalues.
		Values("worker", r.name, "namespace", job.GetNamespace(), "name", job.GetName()).
		Level(4, "job", job).
		Build()...,
	)
	r.recorder.Eventf(jobConfig, corev1.EventTypeNormal, "CreatedJob",
		"Successfully scheduled a new Job: %v", job.GetName())
}

func (r *defaultRecorder) CreateJobFailed(
	_ context.Context,
	jobConfig *execution.JobConfig,
	job *execution.Job,
	message string,
) {
	err := errors.New(message)
	klog.ErrorS(err, "croncontroller: failed to create job", logvalues.
		Values("worker", r.name, "namespace", job.GetNamespace(), "name", job.GetName()).
		Level(4, "job", job).
		Build()...,
	)
	r.recorder.Eventf(jobConfig, corev1.EventTypeWarning, "CreateJobFailed",
		"Failed to create new Job: %v", message)
}

func (r *defaultRecorder) SkippedJobSchedule(
	ctx context.Context,
	jobConfig *execution.JobConfig,
	scheduleTime time.Time,
	message string,
) {
	concurrencyPolicy := jobConfig.Spec.Concurrency.Policy
	klog.V(3).InfoS("croncontroller: skipped job schedule",
		"worker", r.name,
		"namespace", jobConfig.GetNamespace(),
		"name", jobConfig.GetName(),
		"scheduleTime", scheduleTime,
		"concurrencyPolicy", concurrencyPolicy,
		"numActive", jobConfig.Status.Active,
		"numQueued", jobConfig.Status.Queued,
	)
	r.recorder.Event(jobConfig, corev1.EventTypeWarning, "SkippedJobSchedule", message)
}

// newEventRecorder returns a new EventRecorder for the controller.
func newEventRecorder(context controllercontext.Context) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: context.Clientsets().Kubernetes().CoreV1().Events(""),
	})
	return eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})
}
