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

package taskexecutor

import (
	"github.com/pkg/errors"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/argoworkflowtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// Manager is a task executor manager. It holds references to task executor
// factories to create task executors on demand.
type Manager struct {
	podFactory          tasks.ExecutorFactory
	argoWorkflowFactory tasks.ExecutorFactory
}

var _ tasks.ExecutorFactory = (*Manager)(nil)

// NewManager returns a task executor manager that currently always returns the
// Pod task executor.
func NewManager(
	clientsets controllercontext.Clientsets, informers controllercontext.Informers,
) *Manager {
	return &Manager{
		podFactory:          podtaskexecutor.NewFactory(clientsets, informers),
		argoWorkflowFactory: argoworkflowtaskexecutor.NewFactory(clientsets, informers),
	}
}

func (m *Manager) ForJob(job *execution.Job) (tasks.Executor, error) {
	template := job.Spec.Template
	if template == nil {
		return nil, errors.New("job template cannot be empty")
	}
	taskTemplate := template.TaskTemplate

	switch {
	case taskTemplate.Pod != nil:
		return m.podFactory.ForJob(job)
	case taskTemplate.ArgoWorkflow != nil:
		return m.argoWorkflowFactory.ForJob(job)
	}

	return nil, errors.New("no task executors could be matched")
}
