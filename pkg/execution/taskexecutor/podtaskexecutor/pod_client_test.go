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

package podtaskexecutor_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	coreerrors "github.com/furiko-io/furiko/pkg/core/errors"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/errors"
)

const (
	testTimeout = time.Second * 3
)

func TestNewPodTaskClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientset := fake.NewSimpleClientset()
	client := podtaskexecutor.NewClient(fakeJob, clientset.CoreV1().Pods(jobNamespace))

	// Populate Pods
	createdPods := make([]*corev1.Pod, 0, len(fakePods))
	for _, pod := range fakePods {
		createdPod, err := clientset.CoreV1().Pods(pod.GetNamespace()).Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			t.Errorf("cannot create pod: %v", err)
		}
		createdPods = append(createdPods, createdPod)
	}

	// Should be able to get task by index
	task, err := client.Index(ctx, tasks.TaskIndex{Retry: 1})
	assert.NoError(t, err)
	assert.Equal(t, createdPods[0].GetName(), task.GetName())

	// Returns NotFound error when index not found
	_, err = client.Index(ctx, tasks.TaskIndex{Retry: 0})
	assert.True(t, kerrors.IsNotFound(err))
	_, err = client.Index(ctx, tasks.TaskIndex{Retry: 2})
	assert.True(t, kerrors.IsNotFound(err))

	// Should be able to get task by name
	task, err = client.Get(ctx, createdPods[0].GetName())
	assert.NoError(t, err)
	assert.Equal(t, createdPods[0].GetName(), task.GetName())

	// Should also be able to get task that don't belong to job
	task, err = client.Get(ctx, createdPods[2].GetName())
	assert.NoError(t, err)
	assert.Equal(t, createdPods[2].GetName(), task.GetName())

	// Cannot get pods outside of namespace
	_, err = client.Get(ctx, createdPods[1].GetName())
	assert.True(t, kerrors.IsNotFound(err))

	// Returns NotFound error when task not found
	_, err = client.Get(ctx, "invalid-pod")
	assert.True(t, kerrors.IsNotFound(err))

	// Create new index
	newTask, err := client.CreateIndex(ctx, tasks.TaskIndex{Retry: 5})
	assert.NoError(t, err)
	index, ok := newTask.GetRetryIndex()
	assert.True(t, ok)
	assert.Equal(t, int64(5), index)

	// Able to get new task
	task, err = client.Get(ctx, getPodIndexedName(fakeJob.Name, 5))
	assert.NoError(t, err)
	assert.Equal(t, newTask.GetName(), task.GetName())

	// Delete newly created task
	err = client.Delete(ctx, newTask.GetName(), false)
	assert.NoError(t, err)

	// Not idempotent
	err = client.Delete(ctx, newTask.GetName(), false)
	assert.Error(t, err)
}

func TestClient_CreateIndex(t *testing.T) {
	tests := []struct {
		name    string
		job     *execution.Job
		index   tasks.TaskIndex
		reactor *ktesting.SimpleReactor
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "no error",
			job:  fakeJob,
			index: tasks.TaskIndex{
				Retry: 0,
				Parallel: execution.ParallelIndex{
					IndexNumber: pointer.Int64(0),
				},
			},
		},
		{
			name: "Invalid",
			job:  fakeJob,
			index: tasks.TaskIndex{
				Retry: 0,
				Parallel: execution.ParallelIndex{
					IndexNumber: pointer.Int64(0),
				},
			},
			reactor: &ktesting.SimpleReactor{
				Verb:     "*",
				Resource: "*",
				Reaction: func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, kerrors.NewInvalid(core.Kind("pod"), "pod-name", field.ErrorList{})
				},
			},
			wantErr: coreerrors.AssertIsAdmissionRefused(),
		},
		{
			name: "AlreadyExists do not handle as AdmissionRefused",
			job:  fakeJob,
			index: tasks.TaskIndex{
				Retry: 0,
				Parallel: execution.ParallelIndex{
					IndexNumber: pointer.Int64(0),
				},
			},
			reactor: &ktesting.SimpleReactor{
				Verb:     "*",
				Resource: "*",
				Reaction: func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, kerrors.NewAlreadyExists(core.Resource("pod"), "pod-name")
				},
			},
			wantErr: errors.AssertIsFunc("AlreadyExists", kerrors.IsAlreadyExists),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			clientsets := mock.NewClientsets()
			if tt.reactor != nil {
				clientsets.KubernetesMock().PrependReactor(tt.reactor.Verb, tt.reactor.Resource, tt.reactor.React)
			}

			client := podtaskexecutor.NewClient(tt.job, clientsets.Kubernetes().CoreV1().Pods(tt.job.Namespace))
			_, err := client.CreateIndex(ctx, tt.index)
			if errors.Want(t, tt.wantErr, err, "CreateIndex() incorrect error") {
				return
			}
		})
	}
}

func getPodIndexedName(name string, index int64) string {
	name, err := jobutil.GenerateTaskName(fakeJob.Name, tasks.TaskIndex{
		Retry: index,
	})
	if err != nil {
		panic(err)
	}
	return name
}
