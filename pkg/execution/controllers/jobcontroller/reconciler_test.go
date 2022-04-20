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

package jobcontroller_test

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobcontroller"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/meta"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

func TestReconciler(t *testing.T) {
	test := runtimetesting.ReconcilerTest{
		ContextFunc: func(c controllercontext.Context, recorder record.EventRecorder) runtimetesting.ControllerContext {
			return jobcontroller.NewContextWithRecorder(c, recorder)
		},
		ReconcilerFunc: func(c runtimetesting.ControllerContext) reconciler.Reconciler {
			return jobcontroller.NewReconciler(c.(*jobcontroller.Context), runtimetesting.ReconcilerDefaultConcurrency)
		},
		Now: testutils.Mktime(now),
	}

	test.Run(t, []runtimetesting.ReconcilerTestCase{
		{
			Name:   "create pod",
			Target: fakeJob,
			Reactors: runtimetesting.CombinedReactors{
				Kubernetes: []*ktesting.SimpleReactor{
					{
						Verb:     "create",
						Resource: "pods",
						Reaction: func(action ktesting.Action) (bool, runtime.Object, error) {
							// NOTE(irvinlim): Use Reactor to inject a different return value.
							// Here we return a Pod with creationTimestamp added to simulate kube-apiserver.
							return true, fakePodResult.DeepCopy(), nil
						},
					},
				},
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreatePodAction(jobNamespace, fakePod),
					},
				},
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobResult),
					},
				},
			},
		},
		{
			Name:     "do nothing with existing pod and updated result",
			Target:   fakeJobResult,
			Fixtures: []runtime.Object{fakePodResult},
		},
		{
			Name:   "update with pod pending",
			Target: fakeJobResult,
			Fixtures: []runtime.Object{
				fakePodPending,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobPending),
					},
				},
			},
		},
		{
			Name:   "do nothing with updated job status",
			Target: fakeJobPending,
			Fixtures: []runtime.Object{
				fakePodPending,
			},
		},
		{
			Name:   "kill pod with pending timeout",
			Now:    testutils.Mktime(later15m),
			Target: fakeJobResult,
			Fixtures: []runtime.Object{
				fakePodPending,
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							newPod := fakePodPending.DeepCopy()
							meta.SetAnnotation(newPod, podtaskexecutor.LabelKeyKilledFromPendingTimeout, "1")
							return runtimetesting.NewUpdatePodAction(jobNamespace, newPod), nil
						},
						func() (runtimetesting.Action, error) {
							return runtimetesting.NewUpdatePodAction(jobNamespace, fakePodPendingTimeoutTerminating), nil
						},
					},
				},
				Furiko: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							// NOTE(irvinlim): Can only generate JobStatus after the clock is mocked
							object := generateJobStatusFromPod(fakeJobResult, fakePodPendingTimeoutTerminating)
							return runtimetesting.NewUpdateJobStatusAction(jobNamespace, object), nil
						},
					},
				},
			},
		},
		{
			Name:   "do nothing if job has no pending timeout",
			Now:    testutils.Mktime(later15m),
			Target: fakeJobWithoutPendingTimeout,
			Fixtures: []runtime.Object{
				fakePodPending,
			},
		},
		{
			Name:   "do nothing with no default pending timeout",
			Now:    testutils.Mktime(later15m),
			Target: fakeJobPending,
			Fixtures: []runtime.Object{
				fakePodPending,
			},
			Configs: controllercontext.ConfigsMap{
				configv1alpha1.JobExecutionConfigName: &configv1alpha1.JobExecutionConfig{
					DefaultPendingTimeoutSeconds: pointer.Int64(0),
				},
			},
		},
		{
			Name: "do nothing if already marked with pending timeout",
			Now:  testutils.Mktime(later15m),
			TargetGenerator: func() runtime.Object {
				// NOTE(irvinlim): Can only generate JobStatus after the clock is mocked
				return generateJobStatusFromPod(fakeJobResult, fakePodPendingTimeoutTerminating)
			},
			Fixtures: []runtime.Object{
				fakePodPendingTimeoutTerminating,
			},
		},
		{
			Name:   "do nothing if kill timestamp is not yet reached",
			Target: fakeJobWithKillTimestamp,
			Fixtures: []runtime.Object{
				fakePodPending,
			},
		},
		{
			Name:   "kill pod with kill timestamp",
			Now:    testutils.Mktime(killTime),
			Target: fakeJobWithKillTimestamp,
			Fixtures: []runtime.Object{
				fakePodPending,
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdatePodAction(jobNamespace, fakePodTerminating),
					},
				},
			},
		},
		{
			Name:   "do nothing with existing kill timestamp",
			Now:    testutils.Mktime(killTime),
			Target: fakeJobWithKillTimestamp,
			Fixtures: []runtime.Object{
				fakePodTerminating,
			},
		},
		{
			Name: "delete pod with kill timestamp",
			Now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DeleteKillingTasksTimeoutSeconds) * time.Second),
			Target: fakeJobWithKillTimestamp,
			Fixtures: []runtime.Object{
				fakePodTerminating,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobPodDeleting),
					},
				},
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeletePodAction(jobNamespace, fakePod.Name),
					},
				},
			},
		},
		{
			Name: "force delete pod with kill timestamp",
			Now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DeleteKillingTasksTimeoutSeconds) * time.Second).
				Add(time.Duration(*config.DefaultJobExecutionConfig.ForceDeleteKillingTasksTimeoutSeconds) * time.Second),
			Target: fakeJobPodDeleting,
			Fixtures: []runtime.Object{
				fakePodDeleting,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobPodForceDeleting),
					},
				},
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeletePodAction(jobNamespace, fakePod.Name),
					},
				},
			},
		},
		{
			Name: "do not force delete pod if disabled via config",
			Now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DeleteKillingTasksTimeoutSeconds) * time.Second).
				Add(time.Duration(*config.DefaultJobExecutionConfig.ForceDeleteKillingTasksTimeoutSeconds) * time.Second),
			Target: fakeJobPodDeleting,
			Fixtures: []runtime.Object{
				fakePodDeleting,
			},
			Configs: controllercontext.ConfigsMap{
				configv1alpha1.JobExecutionConfigName: &configv1alpha1.JobExecutionConfig{
					ForceDeleteKillingTasksTimeoutSeconds: pointer.Int64(0),
				},
			},
		},
		{
			Name: "do not force delete pod if disabled via JobSpec",
			Now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DeleteKillingTasksTimeoutSeconds) * time.Second).
				Add(time.Duration(*config.DefaultJobExecutionConfig.ForceDeleteKillingTasksTimeoutSeconds) * time.Second),
			Target: fakeJobPodDeletingForbidForceDeletion,
			Fixtures: []runtime.Object{
				fakePodDeleting,
			},
		},
		{
			Name:   "do nothing with already deleted pod",
			Now:    testutils.Mktime(killTime).Add(time.Minute * 3),
			Target: fakeJobPodDeleted,
		},
		{
			Name:   "finalize job",
			Target: fakeJobWithDeletionTimestamp,
			Fixtures: []runtime.Object{
				fakePodTerminating,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					ActionGenerators: []runtimetesting.ActionGenerator{
						func() (runtimetesting.Action, error) {
							// NOTE(irvinlim): Can safely ignore the transient status update here
							action := runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobWithDeletionTimestamp)
							action.IgnoreObject = true
							return action, nil
						},
					},
				},
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeletePodAction(jobNamespace, fakePod.Name),
					},
				},
			},
		},
		{
			Name:   "finalize job with already deleted pods",
			Target: fakeJobWithDeletionTimestampAndKilledPods,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobAction(jobNamespace, fakeJobWithDeletionTimestampAndDeletedPods),
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobWithDeletionTimestampAndDeletedPods),
					},
				},
			},
		},
		{
			Name:   "pod succeeded",
			Now:    testutils.Mktime(later15m),
			Target: fakeJobResult,
			Fixtures: []runtime.Object{
				fakePodFinished,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobFinished),
					},
				},
			},
		},
		{
			Name:   "don't delete finished job on TTL after created/started",
			Now:    testutils.Mktime(later60m),
			Target: fakeJobFinished,
			Fixtures: []runtime.Object{
				fakePodFinished,
			},
		},
		{
			Name: "delete finished job on TTL after finished",
			Now: testutils.Mktime(finishTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DefaultTTLSecondsAfterFinished) * time.Second),
			Target: fakeJobFinished,
			Fixtures: []runtime.Object{
				fakePodFinished,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeleteJobAction(jobNamespace, fakeJob.Name),
					},
				},
			},
		},
		{
			Name:   "delete finished job immediately after finished if set via config",
			Now:    testutils.Mktime(finishTime),
			Target: fakeJobFinished,
			Fixtures: []runtime.Object{
				fakePodFinished,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeleteJobAction(jobNamespace, fakeJob.Name),
					},
				},
			},
			Configs: controllercontext.ConfigsMap{
				configv1alpha1.JobExecutionConfigName: &configv1alpha1.JobExecutionConfig{
					DefaultTTLSecondsAfterFinished: pointer.Int64(0),
				},
			},
		},
		{
			Name:   "delete finished job immediately after finished if set via JobSpec",
			Now:    testutils.Mktime(finishTime),
			Target: fakeJobFinishedWithTTLAfterFinished,
			Fixtures: []runtime.Object{
				fakePodFinished,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeleteJobAction(jobNamespace, fakeJob.Name),
					},
				},
			},
		},
	})
}
