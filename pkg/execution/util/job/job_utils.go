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
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/meta"
)

// AllowedToCreateNewTask returns whether a Job is allowed to create more tasks
// based on TaskRefs. The JobStatus should be up-to-date with all tasks created
// by the Job controller.
//
// It assumes that it should only have at most 1 task running at a time, does
// not create more tasks on success, and stops creating tasks once KillTimestamp
// is set (even if it is in the future).
func AllowedToCreateNewTask(rj *execution.Job) bool {
	// Previously determined cannot create task, so give up.
	if _, ok := GetAdmissionErrorMessage(rj); ok {
		return false
	}

	// There is an active task, or already have a successful task.
	if HasActiveTask(rj) || HasSuccessTask(rj) {
		return false
	}

	// Exceed maximum retries.
	if rj.Status.CreatedTasks >= GetMaxAllowedTasks(rj) {
		return false
	}

	// Job being killed.
	if rj.Spec.KillTimestamp != nil {
		return false
	}

	return true
}

// HasActiveTask returns true if any of the TaskRefs is still active.
// Active means that the task is currently active (pending/running).
func HasActiveTask(rj *execution.Job) bool {
	for _, task := range rj.Status.Tasks {
		if task.FinishTimestamp.IsZero() {
			return true
		}
	}
	return false
}

// HasSuccessTask returns true if any of the TaskRefs are successful.
func HasSuccessTask(rj *execution.Job) bool {
	for _, task := range rj.Status.Tasks {
		if result := task.Status.Result; result != nil && *result == execution.JobResultSuccess {
			return true
		}
	}
	return false
}

// MarkAdmissionError updates a Job to add the AdmissionError annotation.
func MarkAdmissionError(rj *execution.Job, msg string) {
	meta.SetAnnotation(rj, LabelKeyAdmissionErrorMessage, msg)
}

// GetAdmissionErrorMessage returns the error message if the Job contains the
// AdmissionErrorMessage annotation.
func GetAdmissionErrorMessage(rj *execution.Job) (string, bool) {
	val, ok := rj.GetAnnotations()[LabelKeyAdmissionErrorMessage]
	return val, ok
}
