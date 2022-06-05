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

package parallel

import (
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// GetParallelStatus returns the complete ParallelStatus for the Job.
func GetParallelStatus(job *execution.Job, tasks []execution.TaskRef) (execution.ParallelStatus, error) {
	var status execution.ParallelStatus

	template := job.Spec.Template
	if template == nil {
		template = &execution.JobTemplate{}
	}
	spec := template.Parallelism
	if spec == nil {
		spec = &execution.ParallelismSpec{}
	}

	// Compute summary.
	summary, err := GetParallelTaskSummary(job, tasks)
	if err != nil {
		return status, err
	}
	status.ParallelStatusSummary = summary

	indexes := GenerateIndexes(spec)
	hashes, _, err := HashIndexes(indexes)
	if err != nil {
		return status, err
	}

	// Group tasks by index.
	taskIndexes := make(map[string][]execution.TaskRef, len(indexes))
	for _, task := range tasks {
		index := GetDefaultIndex()
		if task.ParallelIndex != nil {
			index = *task.ParallelIndex
		}
		hash, err := HashIndex(index)
		if err != nil {
			return status, errors.Wrapf(err, "cannot hash index")
		}
		taskIndexes[hash] = append(taskIndexes[hash], task)
	}

	// Create status for each index.
	for i, index := range indexes {
		tasks := taskIndexes[hashes[i]]
		status.Indexes = append(status.Indexes, getIndexStatus(index, hashes[i], tasks, job.GetMaxAttempts()))
	}

	return status, nil
}

// GetParallelTaskSummary returns the parallel task summary status of a Job
// according to a list of TaskRef.
func GetParallelTaskSummary(job *execution.Job, tasks []execution.TaskRef) (execution.ParallelStatusSummary, error) {
	var status execution.ParallelStatusSummary

	template := job.Spec.Template
	if template == nil {
		template = &execution.JobTemplate{}
	}
	spec := template.Parallelism
	if spec == nil {
		spec = &execution.ParallelismSpec{}
	}

	// Generate maps for all indexes.
	indexes := GenerateIndexes(spec)
	hashes, _, err := HashIndexes(indexes)
	if err != nil {
		return status, errors.Wrapf(err, "cannot hash indexes")
	}

	// Map all tasks to their parallel index.
	tasksByParallelIndex := make(map[string][]execution.TaskRef, len(indexes))
	for _, task := range tasks {
		parallelIndex := GetDefaultIndex()
		if task.ParallelIndex != nil {
			parallelIndex = *task.ParallelIndex
		}
		hash, err := HashIndex(parallelIndex)
		if err != nil {
			return status, errors.Wrapf(err, "cannot hash index %v", parallelIndex)
		}
		tasksByParallelIndex[hash] = append(tasksByParallelIndex[hash], task)
	}

	// Count number of indexes in each state.
	indexStatuses := make([]execution.ParallelIndexStatus, 0, len(indexes))
	for i := range indexes {
		indexStatuses = append(indexStatuses, getIndexStatus(
			indexes[i],
			hashes[i],
			tasksByParallelIndex[hashes[i]],
			job.GetMaxAttempts(),
		))
	}
	counters := GetParallelStatusCounters(indexStatuses)

	// Compute final success state.
	var successful, failed bool
	switch spec.GetCompletionStrategy() {
	case execution.AllSuccessful:
		successful = counters.Succeeded >= int64(len(indexes))
		failed = counters.Failed > 0
	case execution.AnySuccessful:
		successful = counters.Succeeded > 0
		failed = counters.Failed >= int64(len(indexes))
	}

	// Compute final successful state.
	if successful || failed {
		status.Successful = pointer.Bool(successful)
		status.Complete = true
	}

	return status, nil
}

// GetParallelStatusCounters returns the parallel task summary status of a Job
// according to a list of TaskRef.
func GetParallelStatusCounters(indexes []execution.ParallelIndexStatus) execution.ParallelStatusCounters {
	var status execution.ParallelStatusCounters
	for _, index := range indexes {
		switch index.State {
		case execution.IndexNotCreated:
			status.Terminated++
		case execution.IndexRetryBackoff:
			status.Created++
			status.RetryBackoff++
			status.Terminated++ // Technically all tasks are terminated
		case execution.IndexStarting:
			status.Created++
			status.Starting++
		case execution.IndexRunning:
			status.Created++
			status.Running++
		case execution.IndexTerminated:
			status.Created++
			status.Terminated++
		}

		switch index.Result {
		case execution.TaskSucceeded:
			status.Succeeded++
		case execution.TaskFailed:
			status.Failed++
		}
	}
	return status
}

func getIndexStatus(
	index execution.ParallelIndex,
	hash string,
	tasks []execution.TaskRef,
	maxAttempts int64,
) execution.ParallelIndexStatus {
	var numStarting, numRunning, numTerminal int64
	var succeeded, failed bool

	status := execution.ParallelIndexStatus{
		Index:        index,
		Hash:         hash,
		CreatedTasks: int64(len(tasks)),
	}

	for _, task := range tasks {
		// Count the number of tasks in each state, at most one can be true.
		switch {
		case !task.FinishTimestamp.IsZero():
			numTerminal++
		case !task.RunningTimestamp.IsZero():
			numRunning++
		case task.RunningTimestamp.IsZero():
			numStarting++
		}

		// At least one of the tasks succeeded.
		if task.Status.Result == execution.TaskSucceeded {
			succeeded = true
		}
	}

	// Max attempts has been exceeded.
	if !succeeded && numTerminal >= maxAttempts {
		failed = true
	}

	// Check the current state based on the aggregated sums.
	switch {
	case len(tasks) == 0:
		status.State = execution.IndexNotCreated
	case numTerminal == int64(len(tasks)) && !succeeded && !failed:
		status.State = execution.IndexRetryBackoff
	case numTerminal == int64(len(tasks)):
		status.State = execution.IndexTerminated
	case numRunning > 0:
		status.State = execution.IndexRunning
	case numStarting > 0:
		status.State = execution.IndexStarting
	}

	// Check the result if it is completed.
	switch {
	case succeeded:
		status.Result = execution.TaskSucceeded
	case failed:
		status.Result = execution.TaskFailed
	}

	return status
}
