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

package job

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
)

var (
	// Clock is used to mock time.Now() for tests.
	// Use Clock.Now() instead of time.Now() whenever needed.
	Clock clock.PassiveClock = clock.RealClock{}
)

// GetTaskRef returns the TaskRef given a Task.
func GetTaskRef(existing *execution.TaskRef, task tasks.Task) execution.TaskRef {
	taskRef := task.GetTaskRef()
	finishTime := taskRef.FinishTimestamp
	newTaskRef := taskRef.DeepCopy()

	if existing != nil {
		// Don't clear fields which are set rather than derived.
		newTaskRef.DeletedStatus = existing.DeletedStatus.DeepCopy()

		// Don't clear running or finish timestamps, which could be lost between task updates.
		// NOTE(irvinlim): Our assumption is that once we observe a FinishTimestamp for a task,
		// it will never go back to a running state, so we will simply retain the original timestamp.
		// This seems like an issue with the PodStatus that is generated by Kubelet.
		if newTaskRef.RunningTimestamp.IsZero() {
			newTaskRef.RunningTimestamp = existing.RunningTimestamp
		}
		if newTaskRef.FinishTimestamp.IsZero() {
			newTaskRef.FinishTimestamp = existing.FinishTimestamp
		}
	}

	// TODO(irvinlim): Move this handling into PodTask
	// if taskRef.CannotForceDelete {
	// 	// Overwrite the message if the task is set to forbid force deletion.
	// 	// This will create a less confusing message than a perpetual killing message without a concrete reason.
	// 	taskRef.Status.Message =
	// 		"ForbidForceDeletion - Node is unresponsive, but task forbidden to be force deleted by policy"
	// }

	// Copy Status to DeletedStatus if the TaskRef is finished.
	if !finishTime.IsZero() {
		newTaskRef.DeletedStatus = newTaskRef.Status.DeepCopy()
	}

	return *newTaskRef
}

// UpdateJobTaskRefs will update the TaskRefs in the Job's status given a list of tasks.
// Will return a new copy of the Job whose fields are updated.
func UpdateJobTaskRefs(rj *execution.Job, tasks []tasks.Task) *execution.Job {
	newRj := rj.DeepCopy()
	newRefs := GenerateTaskRefs(rj.Status.Tasks, tasks)
	newRj.Status.Tasks = newRefs
	newRj.Status.CreatedTasks = int64(len(newRefs))
	return newRj
}

// UpdateTaskRefDeletedStatusIfNotSet updates the DeletedStatus for a named Task if not already set.
func UpdateTaskRefDeletedStatusIfNotSet(
	rj *execution.Job, taskName string, status execution.TaskStatus,
) *execution.Job {
	newRj := rj.DeepCopy()
	newTaskRefs := make([]execution.TaskRef, 0, len(newRj.Status.Tasks))
	for _, taskRef := range newRj.Status.Tasks {
		newTaskRef := taskRef.DeepCopy()
		if newTaskRef.Name == taskName && newTaskRef.DeletedStatus == nil {
			newTaskRef.DeletedStatus = status.DeepCopy()
		}
		newTaskRefs = append(newTaskRefs, *newTaskRef)
	}
	newRj.Status.Tasks = newTaskRefs
	return newRj
}

// GenerateTaskRefs reconciles tasks into an existing list of TaskRefs.
// If any task is no longer present, it will transition to TaskDeletedFinalStateUnknown.
func GenerateTaskRefs(existing []execution.TaskRef, tasks []tasks.Task) []execution.TaskRef {
	newRefs := make([]execution.TaskRef, 0, len(tasks)+len(existing))
	newRefNames := make(map[string]struct{}, len(tasks))
	existingRefs := make(map[string]execution.TaskRef, len(existing))

	for _, ref := range existing {
		existingRefs[ref.Name] = ref
	}

	// Convert tasks to TaskRefs
	for _, task := range tasks {
		var existingRef *execution.TaskRef
		if ref, ok := existingRefs[task.GetName()]; ok {
			existingRef = &ref
		}
		ref := GetTaskRef(existingRef, task)
		newRefNames[task.GetName()] = struct{}{}
		newRefs = append(newRefs, ref)
	}

	// Find TaskRefs that were existing before but no longer present in the current list.
	// These are considered to be lost.
	for _, existingRef := range existing {
		if _, ok := newRefNames[existingRef.Name]; !ok {
			newRef := existingRef.DeepCopy()

			// Use current time as finish time if not set.
			if newRef.FinishTimestamp.IsZero() {
				newFinishTime := metav1.NewTime(Clock.Now())
				newRef.FinishTimestamp = &newFinishTime
			}

			// Use DeletedStatus if set.
			// Otherwise, use lost state.
			if existingRef.DeletedStatus != nil {
				newRef.Status = *existingRef.DeletedStatus
			} else {
				newRef.Status.State = execution.TaskDeletedFinalStateUnknown
				if newRef.Status.Result == nil {
					newRef.Status.Result = GetResultPtr(execution.JobResultFinalStateUnknown)
				}
			}

			newRefs = append(newRefs, *newRef)
		}
	}

	// Sort TaskRefs.
	SortTaskRefs(newRefs)

	return newRefs
}

// SortTaskRefs will sort the given TaskRefs by CreationTimestamp in ascending order.
func SortTaskRefs(taskRefs []execution.TaskRef) {
	sort.Slice(taskRefs, func(i, j int) bool {
		ti, tj := taskRefs[i], taskRefs[j]
		if !ti.CreationTimestamp.Equal(&tj.CreationTimestamp) {
			return taskRefs[i].CreationTimestamp.Before(&taskRefs[j].CreationTimestamp)
		}
		return ti.Name < tj.Name
	})
}

// FindTaskRef returns the Task that matches the given Task's name.
func FindTaskRef(rj *execution.Job, task tasks.Task) *execution.TaskRef {
	for _, taskRef := range rj.Status.Tasks {
		if taskRef.Name == task.GetName() {
			return &taskRef
		}
	}
	return nil
}
