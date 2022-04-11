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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobcontroller"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/k8sutils"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

var (
	resourcePod = runtimetesting.NewGroupVersionResource("core", "v1", "pods")
	resourceJob = runtimetesting.NewGroupVersionResource(executiongroup.GroupName, execution.Version, "jobs")
)

func TestReconciler(t *testing.T) {
	tests := []struct {
		name             string
		target           *execution.Job
		targetGenerator  func() *execution.Job
		initialPods      []*corev1.Pod
		wantErr          bool
		now              time.Time
		coreReactors     []*ktesting.SimpleReactor
		coreActions      runtimetesting.ActionTest
		executionActions runtimetesting.ActionTest
		configs          map[configv1alpha1.ConfigName]runtime.Object
	}{
		{
			name:   "create pod",
			target: fakeJob,
			coreActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewCreateAction(resourcePod, jobNamespace, fakePod),
				},
			},
			coreReactors: []*ktesting.SimpleReactor{
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
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateStatusAction(resourceJob, jobNamespace, fakeJobResult),
				},
			},
		},
		{
			name:   "do nothing with existing pod and updated result",
			target: fakeJobResult,
			initialPods: []*corev1.Pod{
				fakePodResult,
			},
		},
		{
			name:   "update with pod pending",
			target: fakeJobResult,
			initialPods: []*corev1.Pod{
				fakePodPending,
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateStatusAction(resourceJob, jobNamespace, fakeJobPending),
				},
			},
		},
		{
			name:   "do nothing with updated job status",
			target: fakeJobPending,
			initialPods: []*corev1.Pod{
				fakePodPending,
			},
		},
		{
			name:   "kill pod with pending timeout",
			now:    testutils.Mktime(later15m),
			target: fakeJobResult,
			initialPods: []*corev1.Pod{
				fakePodPending,
			},
			coreActions: runtimetesting.ActionTest{
				ActionGenerators: []runtimetesting.ActionGenerator{
					func() (runtimetesting.Action, error) {
						newPod := fakePodPending.DeepCopy()
						k8sutils.SetAnnotation(newPod, podtaskexecutor.LabelKeyKilledFromPendingTimeout, "1")
						return runtimetesting.NewUpdateAction(resourcePod, jobNamespace, newPod), nil
					},
					func() (runtimetesting.Action, error) {
						return runtimetesting.NewUpdateAction(resourcePod, jobNamespace, fakePodPendingTimeoutTerminating), nil
					},
				},
			},
			executionActions: runtimetesting.ActionTest{
				ActionGenerators: []runtimetesting.ActionGenerator{
					func() (runtimetesting.Action, error) {
						// NOTE(irvinlim): Can only generate JobStatus after the clock is mocked
						object := generateJobStatusFromPod(fakeJobResult, fakePodPendingTimeoutTerminating)
						return runtimetesting.NewUpdateStatusAction(resourceJob, jobNamespace, object), nil
					},
				},
			},
		},
		{
			name:   "do nothing if job has no pending timeout",
			now:    testutils.Mktime(later15m),
			target: fakeJobWithoutPendingTimeout,
			initialPods: []*corev1.Pod{
				fakePodPending,
			},
		},
		{
			name:   "do nothing with no default pending timeout",
			now:    testutils.Mktime(later15m),
			target: fakeJobPending,
			initialPods: []*corev1.Pod{
				fakePodPending,
			},
			configs: map[configv1alpha1.ConfigName]runtime.Object{
				configv1alpha1.JobExecutionConfigName: &configv1alpha1.JobExecutionConfig{
					DefaultPendingTimeoutSeconds: pointer.Int64(0),
				},
			},
		},
		{
			name: "do nothing if already marked with pending timeout",
			now:  testutils.Mktime(later15m),
			targetGenerator: func() *execution.Job {
				// NOTE(irvinlim): Can only generate JobStatus after the clock is mocked
				return generateJobStatusFromPod(fakeJobResult, fakePodPendingTimeoutTerminating)
			},
			initialPods: []*corev1.Pod{
				fakePodPendingTimeoutTerminating,
			},
		},
		{
			name:   "do nothing if kill timestamp is not yet reached",
			target: fakeJobWithKillTimestamp,
			initialPods: []*corev1.Pod{
				fakePodPending,
			},
		},
		{
			name:   "kill pod with kill timestamp",
			now:    testutils.Mktime(killTime),
			target: fakeJobWithKillTimestamp,
			initialPods: []*corev1.Pod{
				fakePodPending,
			},
			coreActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateAction(resourcePod, jobNamespace, fakePodTerminating),
				},
			},
		},
		{
			name:   "do nothing with existing kill timestamp",
			now:    testutils.Mktime(killTime),
			target: fakeJobWithKillTimestamp,
			initialPods: []*corev1.Pod{
				fakePodTerminating,
			},
		},
		{
			name: "delete pod with kill timestamp",
			now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DeleteKillingTasksTimeoutSeconds) * time.Second),
			target: fakeJobWithKillTimestamp,
			initialPods: []*corev1.Pod{
				fakePodTerminating,
			},
			coreActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewDeleteAction(resourcePod, jobNamespace, fakePod.Name),
				},
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateStatusAction(resourceJob, jobNamespace, fakeJobPodDeleting),
				},
			},
		},
		{
			name: "force delete pod with kill timestamp",
			now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DeleteKillingTasksTimeoutSeconds) * time.Second).
				Add(time.Duration(*config.DefaultJobExecutionConfig.ForceDeleteKillingTasksTimeoutSeconds) * time.Second),
			target: fakeJobPodDeleting,
			initialPods: []*corev1.Pod{
				fakePodDeleting,
			},
			coreActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewDeleteAction(resourcePod, jobNamespace, fakePod.Name),
				},
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateStatusAction(resourceJob, jobNamespace, fakeJobPodForceDeleting),
				},
			},
		},
		{
			name: "do not force delete pod if disabled via config",
			now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DeleteKillingTasksTimeoutSeconds) * time.Second).
				Add(time.Duration(*config.DefaultJobExecutionConfig.ForceDeleteKillingTasksTimeoutSeconds) * time.Second),
			target: fakeJobPodDeleting,
			initialPods: []*corev1.Pod{
				fakePodDeleting,
			},
			configs: map[configv1alpha1.ConfigName]runtime.Object{
				configv1alpha1.JobExecutionConfigName: &configv1alpha1.JobExecutionConfig{
					ForceDeleteKillingTasksTimeoutSeconds: pointer.Int64(0),
				},
			},
		},
		{
			name: "do not force delete pod if disabled via JobSpec",
			now: testutils.Mktime(killTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DeleteKillingTasksTimeoutSeconds) * time.Second).
				Add(time.Duration(*config.DefaultJobExecutionConfig.ForceDeleteKillingTasksTimeoutSeconds) * time.Second),
			target: fakeJobPodDeletingForbidForceDeletion,
			initialPods: []*corev1.Pod{
				fakePodDeleting,
			},
		},
		{
			name:   "do nothing with already deleted pod",
			now:    testutils.Mktime(killTime).Add(time.Minute * 3),
			target: fakeJobPodDeleted,
		},
		{
			name:   "finalize job",
			target: fakeJobWithDeletionTimestamp,
			initialPods: []*corev1.Pod{
				fakePodTerminating,
			},
			coreActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewDeleteAction(resourcePod, jobNamespace, fakePod.Name),
				},
			},
			executionActions: runtimetesting.ActionTest{
				ActionGenerators: []runtimetesting.ActionGenerator{
					func() (runtimetesting.Action, error) {
						// NOTE(irvinlim): Can safely ignore the transient status update here
						action := runtimetesting.NewUpdateStatusAction(resourceJob, jobNamespace, fakeJobWithDeletionTimestamp)
						action.IgnoreObject = true
						return action, nil
					},
				},
			},
		},
		{
			name:   "finalize job with already deleted pods",
			target: fakeJobWithDeletionTimestampAndKilledPods,
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateAction(resourceJob, jobNamespace, fakeJobWithDeletionTimestampAndDeletedPods),
					runtimetesting.NewUpdateStatusAction(resourceJob, jobNamespace, fakeJobWithDeletionTimestampAndDeletedPods),
				},
			},
		},
		{
			name:   "pod succeeded",
			now:    testutils.Mktime(later15m),
			target: fakeJobResult,
			initialPods: []*corev1.Pod{
				fakePodFinished,
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateStatusAction(resourceJob, jobNamespace, fakeJobFinished),
				},
			},
		},
		{
			name:   "don't delete finished job on TTL after created/started",
			now:    testutils.Mktime(later60m),
			target: fakeJobFinished,
			initialPods: []*corev1.Pod{
				fakePodFinished,
			},
		},
		{
			name: "delete finished job on TTL after finished",
			now: testutils.Mktime(finishTime).
				Add(time.Duration(*config.DefaultJobExecutionConfig.DefaultTTLSecondsAfterFinished) * time.Second),
			target: fakeJobFinished,
			initialPods: []*corev1.Pod{
				fakePodFinished,
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewDeleteAction(resourceJob, jobNamespace, fakeJob.Name),
				},
			},
		},
		{
			name:   "delete finished job immediately after finished if set via config",
			now:    testutils.Mktime(finishTime),
			target: fakeJobFinished,
			initialPods: []*corev1.Pod{
				fakePodFinished,
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewDeleteAction(resourceJob, jobNamespace, fakeJob.Name),
				},
			},
			configs: map[configv1alpha1.ConfigName]runtime.Object{
				configv1alpha1.JobExecutionConfigName: &configv1alpha1.JobExecutionConfig{
					DefaultTTLSecondsAfterFinished: pointer.Int64(0),
				},
			},
		},
		{
			name:   "delete finished job immediately after finished if set via JobSpec",
			now:    testutils.Mktime(finishTime),
			target: fakeJobFinishedWithTTLAfterFinished,
			initialPods: []*corev1.Pod{
				fakePodFinished,
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewDeleteAction(resourceJob, jobNamespace, fakeJob.Name),
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			c := mock.NewContext()
			ctrlCtx := jobcontroller.NewContextWithRecorder(c, &record.FakeRecorder{})
			reconciler := jobcontroller.NewReconciler(ctrlCtx, &configv1alpha1.Concurrency{
				Workers: 1,
			})
			c.MockConfigs().SetConfigs(tt.configs)

			err := c.Start(ctx)
			assert.NoError(t, err)
			client := c.MockClientsets()

			// Set the current time
			fakeNow := testutils.Mktime(now)
			if !tt.now.IsZero() {
				fakeNow = tt.now
			}
			fakeClock := clock.NewFakeClock(fakeNow)
			ktime.Clock = fakeClock

			// Create initial objects
			for _, pod := range tt.initialPods {
				pod := pod.DeepCopy()
				_, err := client.Kubernetes().CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("cannot create Pod: %v", err)
				}
			}

			// Create object
			target := tt.target
			if tt.targetGenerator != nil {
				target = tt.targetGenerator()
			}
			target = target.DeepCopy()
			createdJob, err := client.Furiko().ExecutionV1alpha1().Jobs(target.Namespace).
				Create(ctx, target, metav1.CreateOptions{})
			assert.NoError(t, err)

			// NOTE(irvinlim): Add a short delay otherwise cache may not sync consistently
			time.Sleep(time.Millisecond * 10)

			// Wait for cache sync
			if !cache.WaitForCacheSync(ctx.Done(), ctrlCtx.HasSynced...) {
				assert.FailNow(t, "caches not synced")
			}

			// Clear all actions prior to reconcile
			client.ClearActions()

			// Set up reactors
			for _, reactor := range tt.coreReactors {
				client.KubernetesMock().PrependReactor(reactor.Verb, reactor.Resource, reactor.React)
			}

			// Reconcile object
			if err := reconciler.SyncOne(ctx, createdJob.Namespace, createdJob.Name, 0); (err != nil) != tt.wantErr {
				t.Errorf("SyncOne() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Compare actions.
			runtimetesting.CompareActions(t, tt.coreActions, client.KubernetesMock().Actions())
			runtimetesting.CompareActions(t, tt.executionActions, client.FurikoMock().Actions())
		})
	}
}
