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

package taskexecutor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/argoworkflowtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
)

var (
	jobWithNoTemplate = &execution.Job{}

	jobWithNoTaskTemplate = &execution.Job{
		Spec: execution.JobSpec{
			Template: &execution.JobTemplate{},
		},
	}

	podTemplateJob = &execution.Job{
		Spec: execution.JobSpec{
			Template: &execution.JobTemplate{
				TaskTemplate: execution.TaskTemplate{
					Pod: &execution.PodTemplateSpec{},
				},
			},
		},
	}

	workflowTemplateJob = &execution.Job{
		Spec: execution.JobSpec{
			Template: &execution.JobTemplate{
				TaskTemplate: execution.TaskTemplate{
					ArgoWorkflow: &execution.ArgoWorkflowTemplateSpec{},
				},
			},
		},
	}
)

func TestManager(t *testing.T) {
	clientsets := mock.NewClientsets()
	informers := mock.NewInformers(clientsets)
	mgr := taskexecutor.NewManager(clientsets, informers)

	// Requires job template.
	_, err := mgr.ForJob(jobWithNoTemplate)
	assert.Error(t, err)
	_, err = mgr.ForJob(jobWithNoTaskTemplate)
	assert.Error(t, err)

	// Get the correct executor.
	podExecutor, err := mgr.ForJob(podTemplateJob)
	assert.NoError(t, err)
	assert.Equal(t, podtaskexecutor.Kind, podExecutor.GetKind())
	workflowExecutor, err := mgr.ForJob(workflowTemplateJob)
	assert.NoError(t, err)
	assert.Equal(t, argoworkflowtaskexecutor.Kind, workflowExecutor.GetKind())
}
