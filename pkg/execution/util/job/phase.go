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
	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

// GetPhase returns the phase of a Job based on the tasks and Job status so far.
// It expects that the JobCondition is up-to-date.
func GetPhase(rj *v1alpha1.Job) v1alpha1.JobPhase {
	// Handle JobConditionFinished, which is only populated after the Job is terminal.
	if rj.Status.Condition.Finished != nil {
		switch rj.Status.Condition.Finished.Result {
		case v1alpha1.JobResultSuccess:
			return v1alpha1.JobSucceeded
		case v1alpha1.JobResultFailed:
			return v1alpha1.JobFailed
		case v1alpha1.JobResultKilled:
			return v1alpha1.JobKilled
		case v1alpha1.JobResultAdmissionError:
			return v1alpha1.JobAdmissionError
		default:
			return v1alpha1.JobFinishedUnknown
		}
	}

	// Use JobKilling if kill timestamp has passed.
	if ktime.IsTimeSetAndEarlierOrEqual(rj.Spec.KillTimestamp) {
		return v1alpha1.JobKilling
	}

	// Handle JobConditionRunning.
	if rj.Status.Condition.Running != nil {
		if rj.Status.Condition.Running.TerminatingTasks > 0 {
			return v1alpha1.JobTerminating
		}
		return v1alpha1.JobRunning
	}

	// Handle JobConditionWaiting.
	if rj.Status.Condition.Waiting != nil {
		// No tasks created yet.
		if len(rj.Status.Tasks) == 0 {
			return v1alpha1.JobStarting
		}

		parallelStatus := rj.Status.ParallelStatus

		// Special handling for non-parallel job.
		// TODO(irvinlim): Maybe streamline this part
		if parallelStatus == nil {
			latestTask := rj.Status.Tasks[len(rj.Status.Tasks)-1]

			// The latest task has a finish timestamp and it is waiting, which implies that
			// the new task is not yet created.
			if !latestTask.FinishTimestamp.IsZero() {
				return v1alpha1.JobRetryBackoff
			}

			// Otherwise, the latest task is waiting to start. Make a small distinction
			// between Pending and Retrying based on the number of created tasks so far.
			if rj.Status.CreatedTasks > 1 {
				return v1alpha1.JobRetrying
			}

			return v1alpha1.JobPending
		}

		// Some parallel indexes are in retry backoff.
		if parallelStatus.RetryBackoff > 0 {
			return v1alpha1.JobRetryBackoff
		}

		// TODO(irvinlim): Currently cannot differentiate between JobRetrying and
		//  JobPending for parallel jobs.
		return v1alpha1.JobPending
	}

	return v1alpha1.JobQueued
}
