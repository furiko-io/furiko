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

package jobcontroller

import (
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/utils/cmp"
	"github.com/furiko-io/furiko/pkg/utils/meta"
)

// IsJobEqual returns true if the Job is not equal and should be updated.
// This equality check is only true in the context of the JobController.
func IsJobEqual(orig, updated *execution.Job) (bool, error) {
	newUpdated := orig.DeepCopy()
	newUpdated.Spec = updated.Spec
	newUpdated.Annotations = updated.Annotations
	newUpdated.Finalizers = updated.Finalizers
	return cmp.IsJSONEqual(orig, newUpdated)
}

// IsJobStatusEqual returns true if the JobStatus is not equal for a Job.
func IsJobStatusEqual(orig, updated *execution.Job) (bool, error) {
	newUpdated := orig.DeepCopy()
	newUpdated.Status = updated.Status
	return cmp.IsJSONEqual(orig, newUpdated)
}

func isDeleted(rj *execution.Job) bool {
	return !rj.DeletionTimestamp.IsZero()
}

func isFinalized(rj *execution.Job, finalizer string) bool {
	return isDeleted(rj) && !meta.ContainsFinalizer(rj.Finalizers, finalizer)
}

func canCreateTask(rj *execution.Job) bool {
	// Job is being killed.
	if rj.Spec.KillTimestamp != nil {
		return false
	}

	// Stop creating tasks on AdmissionError.
	// TODO(irvinlim): Currently if a single task causes AdmissionError, then the whole job will be terminated.
	if _, ok := job.GetAdmissionErrorMessage(rj); ok {
		return false
	}

	return true
}

func isTaskFinished(task tasks.Task) bool {
	taskStatus := task.GetTaskRef()
	return !taskStatus.FinishTimestamp.IsZero()
}
