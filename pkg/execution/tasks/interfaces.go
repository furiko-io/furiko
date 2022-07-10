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

package tasks

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// Task is implemented by Kubernetes resource definitions that encapsulate a
// single Job task on the cluster. This could be backed by a Pod, etc.
type Task interface {
	// GetOwnerReferences returns the list of owner references for the task.
	// This is used to verify if the task is indeed owned by the Job.
	GetOwnerReferences() []metav1.OwnerReference

	// GetName returns the Task's name.
	GetName() string

	// GetTaskRef returns an immutable copy of the Task.
	GetTaskRef() execution.TaskRef

	// GetKind returns the kind of the task.
	GetKind() string

	// GetRetryIndex returns the retry index for the task.
	// All tasks should be numbered sequentially starting from 1 for a single job.
	GetRetryIndex() (int64, bool)

	// GetParallelIndex returns the parallel index for the task.
	GetParallelIndex() (*execution.ParallelIndex, bool)

	// GetDeletionTimestamp returns the timestamp that the task was requested to be deleted.
	GetDeletionTimestamp() *metav1.Time
}

// TaskIndex contains indexes for a single task.
type TaskIndex struct {
	// Retry refers to the attempt number of a task.
	// Starts from 0.
	Retry int64

	// Parallel refers to the parallel index of a task.
	Parallel execution.ParallelIndex
}

// TaskLister implements methods to list Tasks from informer cache.
type TaskLister interface {
	// Get a single task by name.
	Get(name string) (Task, error)
}

// TaskClient implements methods to perform operations on the apiserver.
type TaskClient interface {
	// CreateIndex creates a new Task with the given index.
	CreateIndex(ctx context.Context, index TaskIndex) (Task, error)

	// Delete will delete the Task with the given name.
	Delete(ctx context.Context, name string, force bool) error
}

// Executor is a task executor interface.
type Executor interface {
	GetKind() string
	Lister() TaskLister
	Client() TaskClient
}

// ExecutorFactory produces task executors.
type ExecutorFactory interface {
	ForJob(job *execution.Job) (Executor, error)
}
