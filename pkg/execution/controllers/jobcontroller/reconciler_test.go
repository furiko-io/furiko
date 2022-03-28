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
	"strconv"
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

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
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
		initialPods      []*corev1.Pod
		wantErr          bool
		now              time.Time
		coreReactors     []*ktesting.SimpleReactor
		coreActions      runtimetesting.ActionTest
		executionActions runtimetesting.ActionTest
	}{
		{
			name:   "create pod",
			target: fakeJob,
			coreActions: runtimetesting.ActionTest{
				Actions: []ktesting.Action{
					ktesting.NewCreateAction(resourcePod, fakeJob.Namespace, fakePod1),
				},
			},
			coreReactors: []*ktesting.SimpleReactor{
				{
					Verb:     "create",
					Resource: "pods",
					Reaction: func(action ktesting.Action) (bool, runtime.Object, error) {
						return true, fakePod1Result.DeepCopy(), nil
					},
				},
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []ktesting.Action{
					ktesting.NewUpdateSubresourceAction(resourceJob, "status", fakeJob.Namespace, fakeJobResult),
				},
			},
		},
		{
			name:   "don't do anything with existing pod and updated result",
			target: fakeJobResult,
			initialPods: []*corev1.Pod{
				fakePod1Result,
			},
		},
		{
			// TODO(irvinlim): Maybe fix reconciler to immediately reconcile updated task
			//  into JobStatus, and reduce 2 API calls into 1.
			name:   "kill pod with pending timeout",
			now:    testutils.Mktime(later15m),
			target: fakeJobResult,
			initialPods: []*corev1.Pod{
				fakePod1Result,
			},
			coreActions: runtimetesting.ActionTest{
				ActionGenerators: []runtimetesting.ActionGenerator{
					func() (ktesting.Action, error) {
						newPod := fakePod1Result.DeepCopy()
						k8sutils.SetAnnotation(newPod, podtaskexecutor.LabelKeyKilledFromPendingTimeout, "1")
						return ktesting.NewUpdateAction(resourcePod, fakeJob.Namespace, newPod), nil
					},
					func() (ktesting.Action, error) {
						newPod := fakePod1Result.DeepCopy()
						k8sutils.SetAnnotation(newPod, podtaskexecutor.LabelKeyKilledFromPendingTimeout, "1")
						k8sutils.SetAnnotation(newPod, podtaskexecutor.LabelKeyTaskKillTimestamp,
							strconv.Itoa(int(testutils.Mktime(later15m).Unix())))
						newPod.Spec.ActiveDeadlineSeconds = testutils.Mkint64p(1)
						return ktesting.NewUpdateAction(resourcePod, fakeJob.Namespace, newPod), nil
					},
				},
			},
		},
		{
			name:   "pod succeeded",
			now:    testutils.Mktime(later15m),
			target: fakeJobResult,
			initialPods: []*corev1.Pod{
				fakePod1Finished,
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []ktesting.Action{
					ktesting.NewUpdateSubresourceAction(resourceJob, "status", fakeJob.Namespace,
						fakeJobFinishedResult),
				},
			},
		},
		{
			name:   "don't delete finished job on TTL after created/started",
			now:    testutils.Mktime(later60m),
			target: fakeJobFinishedResult,
			initialPods: []*corev1.Pod{
				fakePod1Finished,
			},
		},
		{
			name:   "delete finished job on TTL after finished",
			now:    testutils.Mktime(finishAfterTTL),
			target: fakeJobFinishedResult,
			initialPods: []*corev1.Pod{
				fakePod1Finished,
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []ktesting.Action{
					ktesting.NewDeleteAction(resourceJob, fakeJob.Namespace, fakeJob.Name),
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
			reconciler := jobcontroller.NewReconciler(ctrlCtx, &configv1.Concurrency{
				Workers: 1,
			})
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
			createdJob, err := client.Furiko().ExecutionV1alpha1().Jobs(tt.target.Namespace).
				Create(ctx, tt.target, metav1.CreateOptions{})
			assert.NoError(t, err)

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
