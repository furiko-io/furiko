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

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
)

func TestNewPodTaskClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientset := fake.NewSimpleClientset()
	client := podtaskexecutor.NewPodTaskClient(clientset.CoreV1().Pods(jobNamespace), fakeJob)

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
	task, err := client.Index(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, createdPods[0].GetName(), task.GetName())

	// Returns NotFound error when index not found
	_, err = client.Index(ctx, 0)
	assert.True(t, kerrors.IsNotFound(err))
	_, err = client.Index(ctx, 2)
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
	newTask, err := client.CreateIndex(ctx, 5)
	assert.NoError(t, err)
	index, ok := newTask.GetRetryIndex()
	assert.True(t, ok)
	assert.Equal(t, int64(5), index)

	// Able to get new task
	task, err = client.Get(ctx, podtaskexecutor.GetPodIndexedName(fakeJob.Name, 5))
	assert.NoError(t, err)
	assert.Equal(t, newTask.GetName(), task.GetName())

	// Delete newly created task
	err = client.Delete(ctx, newTask.GetName(), false)
	assert.NoError(t, err)

	// Not idempotent
	err = client.Delete(ctx, newTask.GetName(), false)
	assert.Error(t, err)
}
