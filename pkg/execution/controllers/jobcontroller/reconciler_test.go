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
	"k8s.io/apimachinery/pkg/util/clock"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobcontroller"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	flushActionTimeout = time.Millisecond * 100
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
		coreActions      runtimetesting.ActionTest
		executionActions runtimetesting.ActionTest
	}{
		{
			name:   "create pod",
			target: fakeJob,
			coreActions: runtimetesting.ActionTest{
				ActionGenerators: []runtimetesting.ActionGenerator{
					func() (ktesting.Action, error) {
						pod, err := podtaskexecutor.NewPod(fakeJob, 1)
						if err != nil {
							return nil, err
						}
						action := ktesting.NewCreateAction(resourcePod, fakeJob.Namespace, pod)
						return action, nil
					},
				},
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []ktesting.Action{
					ktesting.NewUpdateSubresourceAction(resourceJob, "status", fakeJob.Namespace, fakeJobResult),
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
			ctrlCtx := jobcontroller.NewContextWithCustom(c, &record.FakeRecorder{}, newMockTaskManager(c))
			reconciler := jobcontroller.NewReconciler(ctrlCtx, &configv1.Concurrency{
				Workers: 1,
			})
			err := c.Start(ctx)
			assert.NoError(t, err)

			client := c.MockClientsets()

			// Set the current time
			fakeClock := clock.NewFakeClock(testutils.Mktime(now))
			ktime.Clock = fakeClock

			// Create initial objects
			for _, pod := range tt.initialPods {
				_, err := client.Kubernetes().CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("cannot create Pod: %v", err)
				}
			}

			// Wait for cache sync
			if !cache.WaitForCacheSync(ctx.Done(), ctrlCtx.HasSynced...) {
				assert.FailNow(t, "caches not synced")
			}

			// Create object
			createdJob, err := client.Furiko().ExecutionV1alpha1().Jobs(tt.target.Namespace).
				Create(ctx, tt.target, metav1.CreateOptions{})
			assert.NoError(t, err)

			// Clear all actions prior to reconcile
			client.ClearActions()

			// Reconcile object
			if err := reconciler.SyncOne(ctx, createdJob.Namespace, createdJob.Name, 0); (err != nil) != tt.wantErr {
				t.Errorf("SyncOne() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Sleep for a short duration in case API requests are not yet called
			// TODO(irvinlim): There may be a better way to do this
			time.Sleep(flushActionTimeout)

			// Compare actions.
			runtimetesting.CompareActions(t, tt.coreActions, client.KubernetesMock().Actions())
			runtimetesting.CompareActions(t, tt.executionActions, client.FurikoMock().Actions())
		})
	}
}

// mockTaskManager is a custom ExecutorFactory implementation.
type mockTaskManager struct {
	context controllercontext.ContextInterface
}

func newMockTaskManager(context controllercontext.ContextInterface) *mockTaskManager {
	return &mockTaskManager{context: context}
}

func (m *mockTaskManager) ForJob(rj *execution.Job) (tasks.Executor, error) {
	return &mockTaskExecutor{
		context: m.context,
		rj:      rj,
	}, nil
}

// mockTaskExecutor is a custom TaskExecutor implementation that wraps PodTaskExecutor.
type mockTaskExecutor struct {
	context controllercontext.ContextInterface
	rj      *execution.Job
}

func (m *mockTaskExecutor) Lister() tasks.TaskLister {
	return &mockTaskLister{
		PodTaskLister: podtaskexecutor.NewPodTaskLister(
			m.context.Informers().Kubernetes().Core().V1().Pods().Lister(),
			m.context.Clientsets().Kubernetes().CoreV1().Pods(m.rj.Namespace),
			m.rj,
		),
	}
}

func (m *mockTaskExecutor) Client() tasks.TaskClient {
	return &mockTaskClient{
		PodTaskClient: podtaskexecutor.NewPodTaskClient(
			m.context.Clientsets().Kubernetes().CoreV1().Pods(m.rj.Namespace),
			m.rj,
		),
		rj: m.rj,
	}
}

type mockTaskLister struct {
	*podtaskexecutor.PodTaskLister
}

type mockTaskClient struct {
	*podtaskexecutor.PodTaskClient
	rj *execution.Job
}

func (c *mockTaskClient) CreateIndex(ctx context.Context, index int64) (tasks.Task, error) {
	created, err := c.PodTaskClient.CreateIndex(ctx, index)
	if err != nil {
		return nil, err
	}

	// Wrap into mock task to override certain fields
	return &mockTask{
		Task: created,
		rj:   c.rj,
	}, nil
}

type mockTask struct {
	tasks.Task
	rj *execution.Job
}

func (m *mockTask) GetTaskRef() execution.TaskRef {
	original := m.Task.GetTaskRef()
	newTaskRef := original.DeepCopy()

	// Mock creation timestamp since we don't have a real apiserver.
	// Note that another way to do this could be to use ReactionFunc
	// to intercept the return value of Pod creation.
	newTaskRef.CreationTimestamp = m.rj.CreationTimestamp

	return *newTaskRef
}
