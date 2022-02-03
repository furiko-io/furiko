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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// GetCondition returns a consolidated JobCondition computed from TaskRefs.
func GetCondition(rj *execution.Job) execution.JobCondition {
	state := execution.JobCondition{}

	// We cannot create tasks due to a user error.
	if message, ok := GetAdmissionErrorMessage(rj); ok {
		newStatus := &execution.JobConditionFinished{
			FinishedAt: metav1.NewTime(Clock.Now()),
			Result:     execution.JobResultAdmissionError,
			Reason:     "AdmissionError",
			Message:    message,
		}

		// Use old FinishedAt if previously set.
		if oldStatus := rj.Status.Condition.Finished; oldStatus != nil && !oldStatus.FinishedAt.IsZero() {
			newStatus.FinishedAt = oldStatus.FinishedAt
		}

		state.Finished = newStatus
		return state
	}

	// Not yet started.
	if rj.Status.StartTime.IsZero() {
		var reason, message string

		// The job is explicitly enqueued to start later. If it is nil, it will be
		// started almost immediately so no need to set any special Reason/Message.
		if spec := rj.Spec.StartPolicy; spec != nil {
			if !spec.StartAfter.IsZero() {
				reason = "NotYetDue"
				message = fmt.Sprintf("Job is queued to start no earlier than %v", spec.StartAfter)
			} else if spec.ConcurrencyPolicy == execution.ConcurrencyPolicyEnqueue {
				reason = "Queued"
				message = "Job is queued and may be pending other concurrent jobs to be finished"
			}
		}

		state.Queueing = &execution.JobConditionQueueing{
			Reason:  reason,
			Message: message,
		}
		return state
	}

	// No created tasks yet.
	taskRefs := rj.Status.Tasks
	if len(taskRefs) == 0 {
		// If the job is killed before any tasks are created, we terminate this job.
		if !rj.Spec.KillTimestamp.IsZero() {
			state.Finished = &execution.JobConditionFinished{
				FinishedAt: *rj.Spec.KillTimestamp,
				Result:     execution.JobResultKilled,
			}
			return state
		}

		// Otherwise, the job is waiting for tasks to be created.
		state.Waiting = &execution.JobConditionWaiting{}
		return state
	}

	// Get latest task ref.
	latestTask := taskRefs[len(taskRefs)-1]

	// Latest task is finished.
	if finishTime := latestTask.FinishTimestamp; !finishTime.IsZero() {
		// The task did not succeed, still got more retries.
		if AllowedToCreateNewTask(rj) {
			state.Waiting = &execution.JobConditionWaiting{
				CreatedAt: &latestTask.CreationTimestamp,
				Reason:    "RetryBackoff",
				Message: fmt.Sprintf("Waiting to create new task (retrying %v out of %v)",
					len(taskRefs)+1, GetMaxAllowedTasks(rj),
				),
			}
			return state
		}

		// No more tasks will be created.
		state.Finished = &execution.JobConditionFinished{
			CreatedAt:  &latestTask.CreationTimestamp,
			FinishedAt: *finishTime,
			Reason:     latestTask.Status.Reason,
			Message:    latestTask.Status.Message,
		}
		if result := latestTask.Status.Result; result != nil {
			state.Finished.Result = *result
		}
		if runningTime := latestTask.RunningTimestamp; !runningTime.IsZero() {
			state.Finished.StartedAt = runningTime
		}
		return state
	}

	// Latest task is running.
	if runningTime := latestTask.RunningTimestamp; !runningTime.IsZero() {
		state.Running = &execution.JobConditionRunning{
			CreatedAt: latestTask.CreationTimestamp,
			StartedAt: *runningTime,
		}
		return state
	}

	// Latest task is still waiting.
	createTime := latestTask.CreationTimestamp
	state.Waiting = &execution.JobConditionWaiting{
		CreatedAt: &createTime,
		Reason:    latestTask.Status.Reason,
		Message:   latestTask.Status.Message,
	}
	return state
}
