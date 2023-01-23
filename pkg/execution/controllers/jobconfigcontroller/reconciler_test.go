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

package jobconfigcontroller_test

import (
	"fmt"
	"strconv"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobconfigcontroller"
	"github.com/furiko-io/furiko/pkg/execution/stores/activejobstore"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	createTime1   = "2022-04-01T04:01:00Z"
	createTime2   = "2022-04-01T04:00:00Z"
	startTime     = "2022-04-01T04:01:01Z"
	testNamespace = "test"
	jobConfigUID1 = "0ed1bc76-07ca-4cf7-9a47-a0cc4aec48b9"
	jobConfigUID2 = "6e08ee33-ccbe-4fc5-9c46-e29c19cc2fcb"
	jobUID1       = "9fddf720-f8d3-4773-96af-d483cefddcb7"
	jobUID2       = "f9de7f2e-10e8-414e-b1df-62555aed2213"
)

var (
	jobConfig1 = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-config-1",
			Namespace: testNamespace,
			UID:       jobConfigUID1,
		},
	}

	jobConfig1Ready = makeJobConfig(
		jobConfig1,
		execution.JobConfigReady,
		nil,
		nil,
		nil,
	)

	jobConfig1JobQueued = makeJobConfig(
		jobConfig1,
		execution.JobConfigJobQueued,
		[]func() runtime.Object{job1Queued},
		[]func() runtime.Object{},
		nil,
	)

	jobConfig1Executing = makeJobConfig(
		jobConfig1,
		execution.JobConfigExecuting,
		[]func() runtime.Object{},
		[]func() runtime.Object{job1Running},
		testutils.Mkmtimep(startTime),
	)

	jobConfig1Finished = makeJobConfig(
		jobConfig1,
		execution.JobConfigReady,
		nil,
		nil,
		testutils.Mkmtimep(startTime),
	)

	jobConfig2 = makeJobConfig(&execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-config-2",
			Namespace: testNamespace,
			UID:       jobConfigUID2,
		},
	}, execution.JobConfigReady, nil, nil, testutils.Mkmtimep(startTime))

	ownerReferences = []metav1.OwnerReference{
		{
			APIVersion:         execution.GroupVersion.String(),
			Kind:               execution.KindJobConfig,
			Name:               jobConfig1.Name,
			UID:                jobConfig1.UID,
			Controller:         pointer.Bool(true),
			BlockOwnerDeletion: pointer.Bool(true),
		},
	}

	job1 = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-1",
			Namespace: testNamespace,
			Labels: map[string]string{
				jobconfig.LabelKeyJobConfigUID: jobConfigUID1,
			},
			CreationTimestamp: testutils.Mkmtime(createTime1),
			OwnerReferences:   ownerReferences,
			UID:               jobUID1,
		},
	}
	job1Queued   = makeJob(job1, execution.JobQueued)
	job1Running  = makeJob(job1, execution.JobRunning)
	job1Finished = makeJob(job1, execution.JobSucceeded)

	job2 = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-2",
			Namespace: testNamespace,
			Labels: map[string]string{
				jobconfig.LabelKeyJobConfigUID: jobConfigUID1,
			},
			CreationTimestamp: testutils.Mkmtime(createTime2),
			OwnerReferences:   ownerReferences,
			UID:               jobUID2,
		},
	}
	job2Running = makeJob(job2, execution.JobRunning)

	scheduledJob1 = makeJob(&execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scheduled-job-1",
			Namespace: testNamespace,
			Labels: map[string]string{
				jobconfig.LabelKeyJobConfigUID: jobConfigUID1,
			},
			Annotations: map[string]string{
				jobconfig.AnnotationKeyScheduleTime: strconv.Itoa(int(testutils.Mktime(startTime).Unix())),
			},
			OwnerReferences: ownerReferences,
		},
	}, execution.JobRunning)
)

func TestReconciler(t *testing.T) {
	test := runtimetesting.ReconcilerTest{
		ContextFunc: func(c controllercontext.Context, recorder record.EventRecorder) runtimetesting.ControllerContext {
			return jobconfigcontroller.NewContextWithRecorder(c, recorder)
		},
		ReconcilerFunc: func(c runtimetesting.ControllerContext) reconciler.Reconciler {
			return jobconfigcontroller.NewReconciler(
				c.(*jobconfigcontroller.Context),
				runtimetesting.ReconcilerDefaultConcurrency,
			)
		},
		Stores: []mock.StoreFactory{
			activejobstore.NewFactory(),
		},
		Now: testutils.Mktime(startTime),
	}

	test.Run(t, []runtimetesting.ReconcilerTestCase{
		{
			Name: "no such JobConfig",
			SyncTarget: &runtimetesting.SyncTarget{
				Namespace: testNamespace,
				Name:      "nonexistent-job-config",
			},
		},
		{
			Name:   "update JobConfig status",
			Target: jobConfig1,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobConfigStatusAction(testNamespace, jobConfig1Ready()),
					},
				},
			},
		},
		{
			Name:            "up-to-date JobConfigReady status",
			TargetGenerator: jobConfig1Ready,
		},
		{
			Name:              "update JobConfigReady status for Queued job",
			TargetGenerator:   jobConfig1Ready,
			FixtureGenerators: []func() runtime.Object{job1Queued},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							return runtimetesting.NewUpdateJobConfigStatusAction(testNamespace, jobConfig1JobQueued()), nil
						},
					},
				},
			},
		},
		{
			Name:              "up-to-date JobConfigJobQueued status for Queued job",
			TargetGenerator:   jobConfig1JobQueued,
			FixtureGenerators: []func() runtime.Object{job1Queued},
		},
		{
			Name:              "update JobConfigReady status for Running job",
			TargetGenerator:   jobConfig1Ready,
			FixtureGenerators: []func() runtime.Object{job1Running},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							return runtimetesting.NewUpdateJobConfigStatusAction(testNamespace, jobConfig1Executing()), nil
						},
					},
				},
			},
		},
		{
			Name:              "update JobConfigJobQueued status for Running job",
			TargetGenerator:   jobConfig1JobQueued,
			FixtureGenerators: []func() runtime.Object{job1Running},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							return runtimetesting.NewUpdateJobConfigStatusAction(testNamespace, jobConfig1Executing()), nil
						},
					},
				},
			},
		},
		{
			Name:              "up-to-date JobConfigExecuting status for Running job",
			TargetGenerator:   jobConfig1Executing,
			FixtureGenerators: []func() runtime.Object{job1Running},
		},
		{
			Name:              "update JobConfigExecuting status for Finished job",
			TargetGenerator:   jobConfig1Executing,
			FixtureGenerators: []func() runtime.Object{job1Finished},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							return runtimetesting.NewUpdateJobConfigStatusAction(testNamespace, jobConfig1Finished()), nil
						},
					},
				},
			},
			WantEvents: []runtimetesting.Event{
				{
					UID:     jobConfigUID1,
					Type:    v1.EventTypeNormal,
					Reason:  "Finished",
					Message: fmt.Sprintf("Job %v is complete", job1.Name),
				},
			},
		},
		{
			Name:            "update JobConfigExecuting status for deleted job",
			TargetGenerator: jobConfig1Executing,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							return runtimetesting.NewUpdateJobConfigStatusAction(testNamespace, jobConfig1Finished()), nil
						},
					},
				},
			},
			WantEvents: []runtimetesting.Event{
				{
					UID:     jobConfigUID1,
					Type:    v1.EventTypeNormal,
					Reason:  "Deleted",
					Message: fmt.Sprintf("Job %v is deleted", job1.Name),
				},
			},
		},
		{
			Name:              "no update for JobConfig with non-child jobs",
			TargetGenerator:   jobConfig2,
			FixtureGenerators: []func() runtime.Object{job1Running},
		},
		{
			Name:              "update JobConfigExecuting status for a new Running job",
			TargetGenerator:   jobConfig1Executing,
			FixtureGenerators: []func() runtime.Object{job1Running, job2Running},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							newJobConfig := makeJobConfig(
								jobConfig1,
								execution.JobConfigExecuting,
								[]func() runtime.Object{},
								[]func() runtime.Object{job1Running, job2Running},
								testutils.Mkmtimep(startTime),
							)
							return runtimetesting.NewUpdateJobConfigStatusAction(testNamespace, newJobConfig()), nil
						},
					},
				},
			},
		},
		{
			Name:              "update JobConfigExecuting status with LastScheduleTime",
			Target:            jobConfig1,
			FixtureGenerators: []func() runtime.Object{scheduledJob1},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							newJobConfig := makeJobConfig(
								jobConfig1,
								execution.JobConfigExecuting,
								[]func() runtime.Object{},
								[]func() runtime.Object{scheduledJob1},
								testutils.Mkmtimep(startTime),
							)().(*execution.JobConfig)
							newJobConfig.Status.LastScheduled = testutils.Mkmtimep(startTime)
							return runtimetesting.NewUpdateJobConfigStatusAction(testNamespace, newJobConfig), nil
						},
					},
				},
			},
		},
	})
}

func makeJob(job *execution.Job, phase execution.JobPhase) func() runtime.Object {
	return func() runtime.Object {
		newJob := job.DeepCopy()
		if phase != execution.JobQueued {
			newJob.Status.StartTime = ktime.Now()
		}
		newJob.Status.Phase = phase
		return newJob
	}
}

func makeJobConfig(
	jobConfig *execution.JobConfig,
	state execution.JobConfigState,
	queuedFuncs []func() runtime.Object,
	activeFuncs []func() runtime.Object,
	lastExecuted *metav1.Time,
) func() runtime.Object {
	return func() runtime.Object {
		queued := make([]*execution.Job, 0, len(queuedFuncs))
		active := make([]*execution.Job, 0, len(activeFuncs))
		for _, queuedFunc := range queuedFuncs {
			queued = append(queued, queuedFunc().(*execution.Job))
		}
		for _, activeFunc := range activeFuncs {
			active = append(active, activeFunc().(*execution.Job))
		}

		newJobConfig := jobConfig.DeepCopy()
		newJobConfig.Status.State = state
		newJobConfig.Status.Queued = int64(len(queued))
		newJobConfig.Status.QueuedJobs = jobconfigcontroller.ToJobReferences(queued)
		newJobConfig.Status.Active = int64(len(active))
		newJobConfig.Status.ActiveJobs = jobconfigcontroller.ToJobReferences(active)
		newJobConfig.Status.LastExecuted = lastExecuted
		return newJobConfig
	}
}
