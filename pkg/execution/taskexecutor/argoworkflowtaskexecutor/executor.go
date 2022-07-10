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

package argoworkflowtaskexecutor

import (
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

type executor struct {
	job        *execution.Job
	informers  controllercontext.Informers
	clientsets controllercontext.Clientsets
}

var _ tasks.Executor = (*executor)(nil)

func NewExecutor(
	clientsets controllercontext.Clientsets, informers controllercontext.Informers, job *execution.Job,
) tasks.Executor {
	return &executor{
		clientsets: clientsets,
		informers:  informers,
		job:        job,
	}
}

func (e *executor) Lister() tasks.TaskLister {
	return NewLister(e.job.Namespace, e.informers.Argo().Argoproj().V1alpha1().Workflows().Lister())
}

func (e *executor) Client() tasks.TaskClient {
	return NewClient(e.job, e.clientsets.Argo().ArgoprojV1alpha1().Workflows(e.job.Namespace))
}

type factory struct {
	clientsets controllercontext.Clientsets
	informers  controllercontext.Informers
}

// NewFactory returns a new tasks.ExecutorFactory to return a new task executor.
func NewFactory(clientsets controllercontext.Clientsets, informers controllercontext.Informers) tasks.ExecutorFactory {
	return &factory{
		clientsets: clientsets,
		informers:  informers,
	}
}

func (f *factory) ForJob(job *execution.Job) (tasks.Executor, error) {
	return NewExecutor(f.clientsets, f.informers, job), nil
}
