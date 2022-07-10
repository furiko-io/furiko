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

package argoworkflowtaskexecutor_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/argoworkflowtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
)

func TestNewClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientset := mock.NewClientsets()
	client := argoworkflowtaskexecutor.NewClient(fakeJob, clientset.Argo().ArgoprojV1alpha1().Workflows(jobNamespace))

	// Create new index
	newTask, err := client.CreateIndex(ctx, tasks.TaskIndex{Retry: 1})
	assert.NoError(t, err)
	index, ok := newTask.GetRetryIndex()
	assert.True(t, ok)
	assert.Equal(t, int64(1), index)

	// Able to get new task
	wf, err := clientset.Argo().ArgoprojV1alpha1().Workflows(jobNamespace).Get(ctx, getTaskName(1), metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, newTask.GetName(), wf.Name)

	// Delete newly created task
	err = client.Delete(ctx, newTask.GetName(), false)
	assert.NoError(t, err)

	// Not idempotent
	err = client.Delete(ctx, newTask.GetName(), false)
	assert.Error(t, err)
}

func getTaskName(index int64) string {
	name, err := jobutil.GenerateTaskName(fakeJob.Name, tasks.TaskIndex{
		Retry: index,
	})
	if err != nil {
		panic(err)
	}
	return name
}
