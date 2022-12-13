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
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	coreerrors "github.com/furiko-io/furiko/pkg/core/errors"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/argoworkflowtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/errors"
)

const (
	testTimeout = time.Second * 3
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
					return true, nil, kerrors.NewInvalid(v1alpha1.Kind("workflow"), "workflow-name", field.ErrorList{})
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
					return true, nil, kerrors.NewAlreadyExists(v1alpha1.Resource("workflow"), "workflow-name")
				},
			},
			wantErr: errors.AssertIsFunc("AlreadyExists", kerrors.IsAlreadyExists),
		},
		{
			name: "CRD is NotFound",
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
					return true, nil, kerrors.NewNotFound(core.Resource("customresourcedefinition"), "workflows.argoproj.io")
				},
			},
			wantErr: coreerrors.AssertIsAdmissionRefused(),
		},
		{
			name: "other error",
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
					return true, nil, assert.AnError
				},
			},
			wantErr: errors.AssertAll(
				// Is not AdmissionRefused
				errors.AssertNotIsFunc("AdmissionRefused", coreerrors.IsAdmissionRefused),
				// Is still an error
				assert.Error,
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			clientsets := mock.NewClientsets()
			if tt.reactor != nil {
				clientsets.ArgoMock().PrependReactor(tt.reactor.Verb, tt.reactor.Resource, tt.reactor.React)
			}

			client := argoworkflowtaskexecutor.NewClient(tt.job, clientsets.Argo().ArgoprojV1alpha1().Workflows(tt.job.Namespace))
			_, err := client.CreateIndex(ctx, tt.index)
			if errors.Want(t, tt.wantErr, err, "CreateIndex() incorrect error") {
				return
			}
		})
	}
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
