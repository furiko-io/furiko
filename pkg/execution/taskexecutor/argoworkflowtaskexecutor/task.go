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
	"encoding/json"
	"strconv"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
)

type task struct {
	*v1alpha1.Workflow
}

var _ tasks.Task = (*task)(nil)

func NewTask(wf *v1alpha1.Workflow) tasks.Task {
	return &task{
		Workflow: wf,
	}
}

func (t *task) GetTaskRef() execution.TaskRef {
	task := execution.TaskRef{
		Name:              t.GetName(),
		CreationTimestamp: t.GetCreationTimestamp(),
		Status: execution.TaskStatus{
			State:   t.GetState(),
			Result:  t.GetResult(),
			Reason:  t.GetReason(),
			Message: t.GetMessage(),
		},
	}

	if index, ok := t.GetRetryIndex(); ok {
		task.RetryIndex = index
	}
	if index, ok := t.GetParallelIndex(); ok {
		task.ParallelIndex = index
	}
	if t := t.Status.StartTime(); !t.IsZero() {
		task.RunningTimestamp = t
	}
	if t := t.Status.FinishTime(); !t.IsZero() {
		task.FinishTimestamp = t
	}

	return task
}

func (t *task) GetKind() string {
	return "ArgoWorkflow"
}

func (t *task) GetRetryIndex() (int64, bool) {
	val, ok := t.Workflow.Labels[tasks.LabelKeyTaskRetryIndex]
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}
	return int64(i), true
}

func (t *task) GetParallelIndex() (*execution.ParallelIndex, bool) {
	val, ok := t.Annotations[tasks.AnnotationKeyTaskParallelIndex]
	if !ok {
		return nil, false
	}
	res := &execution.ParallelIndex{}
	if err := json.Unmarshal([]byte(val), res); err != nil {
		return nil, false
	}
	return res, true
}

func (t *task) GetState() execution.TaskState {
	if !t.GetDeletionTimestamp().IsZero() && !t.IsFinished() {
		return execution.TaskKilling
	}

	switch t.Status.Phase {
	case v1alpha1.WorkflowRunning:
		// Note that workflow running state does not imply that pods are already running.
		// Copied the comment from the CRD itself:
		// any node has started; pods might not be running yet, the workflow maybe suspended too
		return execution.TaskRunning
	case v1alpha1.WorkflowError, v1alpha1.WorkflowFailed, v1alpha1.WorkflowSucceeded:
		return execution.TaskTerminated
	case v1alpha1.WorkflowPending:
		fallthrough
	default:
		return execution.TaskStarting
	}
}

func (t *task) IsFinished() bool {
	return t.Status.Phase == v1alpha1.WorkflowFailed ||
		t.Status.Phase == v1alpha1.WorkflowSucceeded ||
		t.Status.Phase == v1alpha1.WorkflowError
}

func (t *task) GetResult() execution.TaskResult {
	switch t.Status.Phase {
	case v1alpha1.WorkflowSucceeded:
		return execution.TaskSucceeded
	case v1alpha1.WorkflowFailed, v1alpha1.WorkflowError:
		return execution.TaskFailed
	}
	return ""
}

func (t *task) GetReason() string {
	// For simplicity, we will return the phase of the Workflow as the reason.
	// This allows the user to quickly identify the current workflow's state.
	return string(t.Status.Phase)
}

func (t *task) GetMessage() string {
	// Directly expose the Workflow's message.
	return t.Status.Message
}
