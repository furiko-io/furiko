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
	"fmt"
	"time"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	jobtasks "github.com/furiko-io/furiko/pkg/execution/tasks"
)

// ContainsActiveTask tests a list of tasks if there are any active tasks. An active
// task is one that is not finished, and is not waiting to be deleted.
func ContainsActiveTask(tasks []jobtasks.Task) bool {
	for _, task := range tasks {
		if IsTaskActive(task) {
			return true
		}
	}
	return false
}

// IsTaskActive tests if the tasks.Task is active. An active task is one that is
// not finished, and is not waiting to be deleted.
func IsTaskActive(task jobtasks.Task) bool {
	return !IsTaskFinished(task) && task.GetDeletionTimestamp() == nil
}

// IsTaskFinished tests if the tasks.Task is finished.
func IsTaskFinished(task jobtasks.Task) bool {
	taskStatus := task.GetTaskRef()
	return !taskStatus.FinishTimestamp.IsZero()
}

// GetMaxAllowedTasks returns the maximum number of allowed tasks that a Job can have.
func GetMaxAllowedTasks(rj *execution.Job) int64 {
	var maxAttempts int64 = 1
	if template := rj.Spec.Template; template != nil {
		if rj.Spec.Template.MaxAttempts != nil {
			maxAttempts = *rj.Spec.Template.MaxAttempts
		}
	}
	return maxAttempts
}

// GetNextAllowedRetry checks if we can create a new task, and if so, returns
// the next time where we are allowed to retry and create a new task, based on
// the Job's Tasks status. If there is no restriction on when the next retry is,
// a zero time will be returned. If not allowed to create a new task, an error
// will be returned.
func GetNextAllowedRetry(rj *execution.Job) (time.Time, error) {
	var nextRetry time.Time
	var retryDelay time.Duration

	// Not allowed to create new task.
	if !AllowedToCreateNewTask(rj) {
		return nextRetry, fmt.Errorf("not allowed to create new task")
	}

	if template := rj.Spec.Template; template != nil {
		// No retry delay set.
		if template.RetryDelaySeconds == nil || *template.RetryDelaySeconds <= 0 {
			return nextRetry, nil
		}
		retryDelay = time.Duration(*template.RetryDelaySeconds) * time.Second
	}

	// No tasks created yet, can start immediately.
	if len(rj.Status.Tasks) == 0 {
		return nextRetry, nil
	}

	// Get latest task ref.
	lastTask := rj.Status.Tasks[len(rj.Status.Tasks)-1]
	finishTime := lastTask.FinishTimestamp

	// The last task is not finished, which should not happen based on the check above.
	if finishTime.IsZero() {
		return nextRetry, fmt.Errorf("last task is not yet finished")
	}

	// Compute next allowed time based on last task's finish timestamp.
	return finishTime.Add(retryDelay), nil
}

// MaxTaskRetryIndex returns the maximum retry index for all tasks.
// Assumes that index starts from 1.
// If there are no tasks, 0 is returned.
func MaxTaskRetryIndex(tasks []jobtasks.Task) int64 {
	var max int64
	for _, task := range tasks {
		index, ok := task.GetRetryIndex()
		if ok {
			if index > max {
				max = index
			}
		}
	}
	return max
}
