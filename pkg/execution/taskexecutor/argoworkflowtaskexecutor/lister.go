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
	argo "github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"

	"github.com/furiko-io/furiko/pkg/execution/tasks"
)

type lister struct {
	namespace string
	informer  argo.WorkflowLister
}

var _ tasks.TaskLister = (*lister)(nil)

func NewLister(namespace string, informer argo.WorkflowLister) tasks.TaskLister {
	return &lister{
		namespace: namespace,
		informer:  informer,
	}
}

func (l *lister) Get(name string) (tasks.Task, error) {
	wf, err := l.informer.Workflows(l.namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return NewTask(wf), nil
}
