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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobcontroller"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
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
				Kubernetes: []*ktesting.SimpleReactor{podCreateReactor},
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreatePodAction(jobNamespace, fakePod0),
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
			Name:   "not allowed to create pod",
			Target: fakeJob,
			Reactors: runtimetesting.CombinedReactors{
				Kubernetes: []*ktesting.SimpleReactor{
					{
						Verb:     "create",
						Resource: "pods",
						Reaction: func(action ktesting.Action) (bool, runtime.Object, error) {
							return true, nil, apierrors.NewForbidden(schema.GroupResource{
								Group:    "v1",
								Resource: "pods",
							}, fakePod0.Name, errors.New("forbidden error"))
						},
					},
				},
			},
			WantError: runtimetesting.AssertErrorIsForbidden(),
			WantEvents: []runtimetesting.Event{
				{
					Type:    corev1.EventTypeWarning,
					Reason:  "CreateTaskError",
					Message: "Error creating Pod: could not create pod: pods.v1 \"my-sample-job-gezdqo-0\" is forbidden: forbidden error",
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
			Name:   "delete pod after pending timeout",
			Now:    testutils.Mktime(later15m),
			Target: fakeJobResult,
			Fixtures: []runtime.Object{
				fakePodPending,
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeletePodAction(jobNamespace, fakePod0.Name),
					},
				},
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobWithPodInPendingTimeout),
					},
				},
			},
		},
		{
			Name:   "update with deleting pod after pending timeout",
			Now:    testutils.Mktime(later15m),
			Target: fakeJobWithPodInPendingTimeout,
			Fixtures: []runtime.Object{
				fakePodPendingDeleting,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobWithPodInPendingTimeoutAfterUpdate),
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
			Name:   "do nothing with disabled default pending timeout",
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
			Name:   "do nothing if already deleting",
			Now:    testutils.Mktime(later15m),
			Target: fakeJobWithPodInPendingTimeoutAfterUpdate,
			Fixtures: []runtime.Object{
				fakePodPendingDeleting,
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
			Name:   "delete pod when job is being killed",
			Now:    testutils.Mktime(killTime),
			Target: fakeJobWithKillTimestamp,
			Fixtures: []runtime.Object{
				fakePodPending,
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeletePodAction(jobNamespace, fakePod0.Name),
					},
				},
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobWithPodInKilling),
					},
				},
			},
		},
		{
			Name:   "update with deleting pod when job is being killed",
			Now:    testutils.Mktime(killTime),
			Target: fakeJobWithPodInKilling,
			Fixtures: []runtime.Object{
				fakePodPendingDeleting,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobWithPodInKillingAfterUpdate),
					},
				},
			},
		},
		{
			Name:   "do nothing with already deleting pod when job is being killed",
			Now:    testutils.Mktime(killTime),
			Target: fakeJobWithPodInKillingAfterUpdate,
			Fixtures: []runtime.Object{
				fakePodPendingDeleting,
			},
		},
		{
			Name:   "do nothing with already deleting pod when job is being killed after some time passes",
			Now:    testutils.Mktime(killTime).Add(time.Second),
			Target: fakeJobWithPodInKillingAfterUpdate,
			Fixtures: []runtime.Object{
				fakePodPendingDeleting,
			},
		},
		{
			Name: "force delete pod when job is being killed",
			Now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.ForceDeleteTaskTimeoutSeconds) * time.Second),
			Target: fakeJobPodDeleting,
			Fixtures: []runtime.Object{
				fakePodPendingDeleting,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobPodForceDeleting),
					},
				},
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeletePodAction(jobNamespace, fakePod0.Name),
					},
				},
			},
		},
		{
			Name: "do not force delete pod if disabled via config",
			Now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.ForceDeleteTaskTimeoutSeconds) * time.Second),
			Target: fakeJobPodDeleting,
			Fixtures: []runtime.Object{
				fakePodPendingDeleting,
			},
			Configs: controllercontext.ConfigsMap{
				configv1alpha1.JobExecutionConfigName: &configv1alpha1.JobExecutionConfig{
					ForceDeleteTaskTimeoutSeconds: pointer.Int64(0),
				},
			},
		},
		{
			Name: "do not force delete pod if disabled via JobSpec",
			Now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.ForceDeleteTaskTimeoutSeconds) * time.Second),
			Target: fakeJobPodDeletingForbidForceDeletion,
			Fixtures: []runtime.Object{
				fakePodPendingDeleting,
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
				fakePodPendingDeleting,
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
						runtimetesting.NewDeletePodAction(jobNamespace, fakePod0.Name),
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

		// Parallelism test cases
		{
			Name:   "parallel: create multiple pods",
			Target: fakeJobParallel,
			Reactors: runtimetesting.CombinedReactors{
				Kubernetes: []*ktesting.SimpleReactor{podCreateReactor},
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreatePodAction(jobNamespace, fakePod0),
						runtimetesting.NewCreatePodAction(jobNamespace, fakePod1),
						runtimetesting.NewCreatePodAction(jobNamespace, fakePod2),
					},
				},
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobParallelResult),
					},
				},
			},
		},
		{
			Name: "parallel: create remaining uncreated pods",
			Target: generateJobStatusFromPod(
				fakeJobParallel,
				podCreated(fakePod0),
				podCreated(fakePod1),
			),
			Reactors: runtimetesting.CombinedReactors{
				Kubernetes: []*ktesting.SimpleReactor{podCreateReactor},
			},
			Fixtures: []runtime.Object{
				podCreated(fakePod0),
				podCreated(fakePod1),
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreatePodAction(jobNamespace, fakePod2),
					},
				},
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobParallelResult),
					},
				},
			},
		},
		{
			Name: "parallel: adopt already existing pods but not in status",
			Target: generateJobStatusFromPod(
				fakeJobParallel,
				podCreated(fakePod0),
				podCreated(fakePod1),
			),
			Fixtures: []runtime.Object{
				podCreated(fakePod0),
				podCreated(fakePod1),
				podCreated(fakePod2),
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreatePodAction(jobNamespace, fakePod2),
					},
				},
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobParallelResult),
					},
				},
			},
		},
		{
			Name:   "parallel: do nothing with all pods created and updated result",
			Target: fakeJobParallelResult,
			Fixtures: []runtime.Object{
				podCreated(fakePod0),
				podCreated(fakePod1),
				podCreated(fakePod2),
			},
		},
		{
			Name: "parallel: delay retry creating next task after failure",
			Now:  testutils.Mktime(finishTime),
			Target: generateJobStatusFromPod(
				withRetryDelay(withMaxAttempts(fakeJobParallel, 2), time.Minute),
				podFinished(fakePod0),
				podFinished(fakePod1),
				podFailed(fakePod2)),
			Fixtures: []runtime.Object{
				podFinished(fakePod0),
				podFinished(fakePod1),
				podFailed(fakePod2),
			},
		},
		{
			Name: "parallel: retry creating next task after failure and delay",
			Now:  testutils.Mktime(retryTime),
			Reactors: runtimetesting.CombinedReactors{
				Kubernetes: []*ktesting.SimpleReactor{newPodCreateReactor(testutils.Mkmtime(retryTime))},
			},
			Target: fakeJobParallelDelayingRetry,
			Fixtures: []runtime.Object{
				podFinished(fakePod0),
				podFinished(fakePod1),
				podFailed(fakePod2),
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreatePodAction(jobNamespace, fakePod21),
					},
				},
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobParallelRetried),
					},
				},
			},
		},
		{
			Name:   "parallel: stop creating tasks after retry has succeeded",
			Now:    testutils.Mktime(retryTime),
			Target: fakeJobParallelRetried,
			Fixtures: []runtime.Object{
				podFinished(fakePod0),
				podFinished(fakePod1),
				podFailed(fakePod2),
				podFinished(fakePod21),
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace,
							generateJobStatusFromPod(fakeJobParallelRetried, podFinished(fakePod21))),
					},
				},
			},
		},
		{
			Name:   "parallel: delete all remaining pods on single index failure with AllSuccessful",
			Target: fakeJobParallelResult,
			Fixtures: []runtime.Object{
				podCreated(fakePod0),
				podCreated(fakePod1),
				podFailed(fakePod2),
			},
			WantActions: runtimetesting.CombinedActions{
				Kubernetes: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewDeletePodAction(jobNamespace, fakePod1.Name),
						runtimetesting.NewDeletePodAction(jobNamespace, fakePod0.Name),
					},
				},
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, fakeJobParallelWithDeletedFailures),
					},
				},
			},
		},
		{
			Name:   "parallel: do not delete pods on single index failure with AnySuccessful",
			Target: withCompletionStrategy(fakeJobParallelResult, execution.AnySuccessful),
			Fixtures: []runtime.Object{
				podCreated(fakePod0),
				podCreated(fakePod1),
				podFailed(fakePod2),
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, generateJobStatusFromPod(
							withCompletionStrategy(fakeJobParallelResult, execution.AnySuccessful),
							podCreated(fakePod0),
							podCreated(fakePod1),
							podFailed(fakePod2),
						)),
					},
				},
			},
		},
	})
}
