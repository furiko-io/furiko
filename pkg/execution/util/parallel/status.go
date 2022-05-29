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

// GetParallelStatus returns the completion status of a Job according to a list of TaskRef.
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
	for i := range indexes {
		index := getIndexStatus(tasksByParallelIndex[hashes[i]], job.GetMaxAttempts())
		if index.created {
			status.Created++
		}
		if index.starting {
			status.Starting++
		}
		if index.running {
			status.Running++
		}
		if index.backoff {
			status.RetryBackoff++
		}
		if index.terminated {
			status.Terminated++
		}
		if index.succeeded {
			status.Succeeded++
		}
		if index.failed {
			status.Failed++
		}
	}

	// Compute final success state.
	var successful, failed bool
	switch spec.GetCompletionStrategy() {
	case execution.AllSuccessful:
		successful = status.Succeeded >= int64(len(indexes))
		failed = status.Failed > 0
	case execution.AnySuccessful:
		successful = status.Succeeded > 0
		failed = status.Failed >= int64(len(indexes))
	}

	// Compute final successful state.
	if successful || failed {
		status.Successful = pointer.Bool(successful)
		status.Complete = true
	}

	return status, nil
}

type indexStatus struct {
	created    bool
	backoff    bool
	starting   bool
	running    bool
	terminated bool
	succeeded  bool
	failed     bool
}

func getIndexStatus(tasks []execution.TaskRef, maxAttempts int64) indexStatus {
	var status indexStatus
	var numStarting, numRunning, numTerminal int64

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

		if task.Status.Result == execution.TaskSucceeded {
			status.succeeded = true
		}
	}
	if !status.succeeded && numTerminal >= maxAttempts {
		status.failed = true
	}

	// Track if there is at least one created task for this index.
	if len(tasks) > 0 {
		status.created = true
	}

	// Track if all tasks that were created are terminated, including when no tasks
	// are created.
	if numTerminal == int64(len(tasks)) {
		status.terminated = true
	}

	// Track statuses.
	if !status.succeeded && !status.failed {
		switch {
		case numRunning > 0:
			status.running = true
		case numStarting > 0:
			status.starting = true
		case len(tasks) > 0:
			status.backoff = true
		}
	}

	return status
}
