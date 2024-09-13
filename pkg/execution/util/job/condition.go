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

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/parallel"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

// GetCondition returns a consolidated JobCondition computed from TaskRefs.
// nolint: gocognit
func GetCondition(rj *execution.Job) (execution.JobCondition, error) {
	state := execution.JobCondition{}

	// We cannot create tasks due to a user error.
	if message, ok := GetAdmissionErrorMessage(rj); ok {
		newStatus := &execution.JobConditionFinished{
			FinishTimestamp: *ktime.Now(),
			Result:          execution.JobResultAdmissionError,
			Reason:          "AdmissionError",
			Message:         message,
		}

		// Use old FinishTimestamp if previously set.
		if oldStatus := rj.Status.Condition.Finished; oldStatus != nil && !oldStatus.FinishTimestamp.IsZero() {
			newStatus.FinishTimestamp = oldStatus.FinishTimestamp
		}

		state.Finished = newStatus
		return state, nil
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
		return state, nil
	}

	template := rj.Spec.Template
	if template == nil {
		template = &execution.JobTemplate{}
	}
	indexes := parallel.GenerateIndexes(template.Parallelism)
	numIndexes := int64(len(indexes))

	// Get parallel status.
	parallelStatus, err := parallel.GetParallelStatus(rj, rj.Status.Tasks)
	if err != nil {
		return state, errors.Wrapf(err, "cannot compute parallel status")
	}
	counters := parallel.GetParallelStatusCounters(parallelStatus.Indexes)

	// Compute latest timestamps across all tasks.
	var latestCreated, latestRunning, latestFinished *metav1.Time
	for _, task := range rj.Status.Tasks {
		latestCreated = ktime.TimeMax(latestCreated, &task.CreationTimestamp)
		latestRunning = ktime.TimeMax(latestRunning, task.RunningTimestamp)
		latestFinished = ktime.TimeMax(latestFinished, task.FinishTimestamp)
	}

	// Job is being killed.
	if ktime.IsTimeSetAndEarlierOrEqual(rj.Spec.KillTimestamp) {
		// All tasks are already terminated.
		if counters.Terminated >= numIndexes {
			finishTimestamp := rj.Spec.KillTimestamp
			if !latestFinished.IsZero() {
				finishTimestamp = latestFinished
			}
			state.Finished = &execution.JobConditionFinished{
				LatestCreationTimestamp: latestCreated,
				LatestRunningTimestamp:  latestRunning,
				FinishTimestamp:         *finishTimestamp,
				Result:                  execution.JobResultKilled,
			}
			return state, nil
		}

		// Otherwise, use Waiting state and show how many more tasks are not yet terminated.
		numNotDeleted := numIndexes - counters.Terminated
		state.Waiting = &execution.JobConditionWaiting{
			Reason:  "DeletingTasks",
			Message: fmt.Sprintf("Waiting for %v out of %v task(s) to be deleted", numNotDeleted, numIndexes),
		}
		return state, nil
	}

	// If not complete, it must be either waiting or running.
	if !parallelStatus.Complete {
		// Not complete and created indexes is less than expected.
		if counters.Created < numIndexes {
			state.Waiting = &execution.JobConditionWaiting{
				Reason: "PendingCreation",
				Message: fmt.Sprintf("Waiting for %v out of %v task(s) to be created",
					numIndexes-counters.Created, len(indexes)),
			}
			return state, nil
		}

		// Some tasks are in retry backoff.
		if counters.RetryBackoff > 0 {
			state.Waiting = &execution.JobConditionWaiting{
				Reason:  "RetryBackoff",
				Message: fmt.Sprintf("Waiting to retry creating %v task(s)", counters.RetryBackoff),
			}
			return state, nil
		}

		// Not complete and running indexes is less than expected.
		if counters.Starting > 0 {
			state.Waiting = &execution.JobConditionWaiting{
				Reason:  "WaitingForTasks",
				Message: fmt.Sprintf("Waiting for %v task(s) to start running", counters.Starting),
			}
			return state, nil
		}

		// The job is currently running.
		state.Running = &execution.JobConditionRunning{}
		if !latestCreated.IsZero() {
			state.Running.LatestCreationTimestamp = *latestCreated
		}
		if !latestRunning.IsZero() {
			state.Running.LatestRunningTimestamp = *latestRunning
		}
		return state, nil
	}

	// The job is complete but currently waiting to be terminated.
	if counters.Terminated < numIndexes {
		state.Running = &execution.JobConditionRunning{
			TerminatingTasks: numIndexes - counters.Terminated,
		}
		if !latestCreated.IsZero() {
			state.Running.LatestCreationTimestamp = *latestCreated
		}
		if !latestRunning.IsZero() {
			state.Running.LatestRunningTimestamp = *latestRunning
		}
		return state, nil
	}

	// The job is now completely finished.
	state.Finished = &execution.JobConditionFinished{
		LatestCreationTimestamp: latestCreated,
		LatestRunningTimestamp:  latestRunning,
	}
	if !latestFinished.IsZero() {
		state.Finished.FinishTimestamp = *latestFinished
	}

	// Compute finished result.
	state.Finished.Result = execution.JobResultFinalStateUnknown
	if !rj.Spec.KillTimestamp.IsZero() {
		state.Finished.Result = execution.JobResultKilled
	} else if parallelStatus.Successful != nil {
		if *parallelStatus.Successful {
			state.Finished.Result = execution.JobResultSuccess
		} else {
			state.Finished.Result = execution.JobResultFailed
		}
	}

	return state, nil
}
