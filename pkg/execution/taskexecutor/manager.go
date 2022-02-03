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
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// Manager is a task executor manager. It holds references to task executor
// factories to create task executors on demand.
type Manager struct {
	tasks.ExecutorFactory
}

// NewManager returns a task executor manager that currently always returns the
// Pod task executor.
func NewManager(
	clientsets controllercontext.Clientsets, informers controllercontext.Informers,
) *Manager {
	return &Manager{
		ExecutorFactory: podtaskexecutor.NewFactory(clientsets, informers),
	}
}
