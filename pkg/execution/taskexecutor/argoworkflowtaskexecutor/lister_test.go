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

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/argoworkflowtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
)

var (
	fakeWorkflows = []*v1alpha1.Workflow{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getTaskName(0),
				Namespace: jobNamespace,
				Labels: map[string]string{
					tasks.LabelKeyJobUID: jobUID,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalid",
				Namespace: "invalid",
				Labels: map[string]string{
					tasks.LabelKeyJobUID: "invalid",
				},
			},
		},
	}
)

func TestNewLister(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientset := mock.NewClientsets()
	informerFactory := mock.NewInformers(clientset).Argo()
	lister := argoworkflowtaskexecutor.NewLister(jobNamespace, informerFactory.Argoproj().V1alpha1().Workflows().Lister())

	// Populate Workflows
	createdWorkflows := make([]*v1alpha1.Workflow, 0, len(fakeWorkflows))
	for _, wf := range fakeWorkflows {
		createdWf, err := clientset.Argo().ArgoprojV1alpha1().Workflows(wf.GetNamespace()).Create(ctx, wf, metav1.CreateOptions{})
		if err != nil {
			t.Errorf("cannot create workflow: %v", err)
		}
		createdWorkflows = append(createdWorkflows, createdWf)
	}

	// Start informer
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())

	// Should be able to get task by name
	task, err := lister.Get(getTaskName(0))
	assert.NoError(t, err)
	assert.Equal(t, createdWorkflows[0].GetName(), task.GetName())

	// Returns NotFound error when index not found
	_, err = lister.Get(getTaskName(2))
	assert.True(t, kerrors.IsNotFound(err))

	// Should be able to get task by name
	task, err = lister.Get(createdWorkflows[0].GetName())
	assert.NoError(t, err)
	assert.Equal(t, createdWorkflows[0].GetName(), task.GetName())

	// Cannot get tasks outside of namespace
	_, err = lister.Get(createdWorkflows[1].GetName())
	assert.True(t, kerrors.IsNotFound(err))

	// Returns NotFound error when task not found
	_, err = lister.Get("invalid")
	assert.True(t, kerrors.IsNotFound(err))
}
