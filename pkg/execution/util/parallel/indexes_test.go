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

package parallel_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/parallel"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	createTime = "2022-04-01T04:01:00Z"
	runTime    = "2022-04-01T04:01:05Z"
	finishTime = "2022-04-01T04:01:10Z"
)

func ExampleHashIndex() {
	hash, err := parallel.HashIndex(parallel.GetDefaultIndex())
	if err != nil {
		panic(err)
	}
	fmt.Println(hash)
	// Output: gezdqo
}

func TestHashTaskParallelIndex(t *testing.T) {
	hash := func(index execution.ParallelIndex) string {
		hash, err := parallel.HashIndex(index)
		assert.NoError(t, err)
		return hash
	}

	// By IndexNumber
	assert.Equal(t, hash(execution.ParallelIndex{
		IndexNumber: pointer.Int64(0),
	}), hash(execution.ParallelIndex{
		IndexNumber: pointer.Int64(0),
	}))
	assert.NotEqual(t, hash(execution.ParallelIndex{
		IndexNumber: pointer.Int64(0),
	}), hash(execution.ParallelIndex{
		IndexNumber: pointer.Int64(1),
	}))

	// By IndexKey
	assert.Equal(t, hash(execution.ParallelIndex{
		IndexKey: "key",
	}), hash(execution.ParallelIndex{
		IndexKey: "key",
	}))
	assert.NotEqual(t, hash(execution.ParallelIndex{
		IndexKey: "key",
	}), hash(execution.ParallelIndex{
		IndexKey: "diff",
	}))

	// By MatrixValues
	assert.Equal(t, hash(execution.ParallelIndex{
		MatrixValues: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}), hash(execution.ParallelIndex{
		MatrixValues: map[string]string{
			"key2": "value2",
			"key1": "value1",
		},
	}))
	assert.NotEqual(t, hash(execution.ParallelIndex{
		MatrixValues: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}), hash(execution.ParallelIndex{
		MatrixValues: map[string]string{
			"key1": "value1",
			"key2": "value3",
		},
	}))
}

func TestDiffMissingIndexes(t *testing.T) {
	type args struct {
		job     *execution.Job
		indexes []execution.ParallelIndex
	}
	tests := []struct {
		name    string
		args    args
		want    []parallel.IndexCreationRequest
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "no indexes",
			args: args{
				job: &execution.Job{},
			},
			want: []parallel.IndexCreationRequest{},
		},
		{
			name: "no task statuses",
			args: args{
				job: &execution.Job{},
				indexes: []execution.ParallelIndex{
					{IndexNumber: pointer.Int64(0)},
				},
			},
			want: []parallel.IndexCreationRequest{
				{
					ParallelIndex: execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
					RetryIndex:    0,
				},
			},
		},
		{
			name: "active task status",
			args: args{
				job: &execution.Job{
					Status: execution.JobStatus{
						Tasks: []execution.TaskRef{
							makeTask("task-1", 0, 0, execution.TaskRunning, ""),
						},
					},
				},
				indexes: []execution.ParallelIndex{
					{IndexNumber: pointer.Int64(0)},
				},
			},
			want: []parallel.IndexCreationRequest{},
		},
		{
			name: "failed task status without retry",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{},
					},
					Status: execution.JobStatus{
						Tasks: []execution.TaskRef{
							makeTask("task-1", 0, 0, execution.TaskTerminated, execution.TaskFailed),
						},
					},
				},
				indexes: []execution.ParallelIndex{
					{IndexNumber: pointer.Int64(0)},
				},
			},
			want: []parallel.IndexCreationRequest{},
		},
		{
			name: "failed task status with retry",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(3),
						},
					},
					Status: execution.JobStatus{
						Tasks: []execution.TaskRef{
							makeTask("task-1", 0, 0, execution.TaskTerminated, execution.TaskFailed),
						},
					},
				},
				indexes: []execution.ParallelIndex{
					{IndexNumber: pointer.Int64(0)},
				},
			},
			want: []parallel.IndexCreationRequest{
				{
					ParallelIndex: execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
					RetryIndex:    1,
					Earliest:      testutils.Mktime(finishTime),
				},
			},
		},
		{
			name: "some task statuses are successful with retry",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(3),
						},
					},
					Status: execution.JobStatus{
						Tasks: []execution.TaskRef{
							makeTask("task-1", 0, 0, execution.TaskTerminated, execution.TaskSucceeded),
							makeTask("task-2", 1, 0, execution.TaskTerminated, execution.TaskFailed),
						},
					},
				},
				indexes: []execution.ParallelIndex{
					{IndexNumber: pointer.Int64(0)},
					{IndexNumber: pointer.Int64(1)},
				},
			},
			want: []parallel.IndexCreationRequest{
				{
					ParallelIndex: execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
					RetryIndex:    1,
					Earliest:      testutils.Mktime(finishTime),
				},
			},
		},
		{
			name: "retry delay",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts:       pointer.Int64(3),
							RetryDelaySeconds: pointer.Int64(60),
						},
					},
					Status: execution.JobStatus{
						Tasks: []execution.TaskRef{
							makeTask("task-1", 0, 0, execution.TaskTerminated, execution.TaskFailed),
						},
					},
				},
				indexes: []execution.ParallelIndex{
					{IndexNumber: pointer.Int64(0)},
				},
			},
			want: []parallel.IndexCreationRequest{
				{
					ParallelIndex: execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
					RetryIndex:    1,
					Earliest:      testutils.Mktime(finishTime).Add(time.Second * 60),
				},
			},
		},
		{
			name: "exceed max retries",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(3),
						},
					},
					Status: execution.JobStatus{
						Tasks: []execution.TaskRef{
							makeTask("task-1-attempt-0", 0, 0, execution.TaskTerminated, execution.TaskSucceeded),
							makeTask("task-2-attempt-0", 1, 0, execution.TaskTerminated, execution.TaskFailed),
							makeTask("task-2-attempt-1", 1, 1, execution.TaskTerminated, execution.TaskFailed),
							makeTask("task-2-attempt-2", 1, 2, execution.TaskTerminated, execution.TaskFailed),
						},
					},
				},
				indexes: []execution.ParallelIndex{
					{IndexNumber: pointer.Int64(0)},
					{IndexNumber: pointer.Int64(1)},
				},
			},
			want: []parallel.IndexCreationRequest{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parallel.ComputeMissingIndexesForCreation(tt.args.job, tt.args.indexes)
			if testutils.WantError(t, tt.wantErr, err,
				fmt.Sprintf("ComputeMissingIndexesForCreation(%v, %v)", tt.args.job, tt.args.indexes)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ComputeMissingIndexesForCreation(%v, %v)", tt.args.job, tt.args.indexes)
		})
	}
}

func makeTask(
	name string,
	parallelIndex, retryIndex int64,
	state execution.TaskState,
	result execution.TaskResult,
) execution.TaskRef {
	task := execution.TaskRef{
		Name:              name,
		CreationTimestamp: testutils.Mkmtime(createTime),
		RunningTimestamp:  testutils.Mkmtimep(runTime),
		RetryIndex:        retryIndex,
		ParallelIndex: &execution.ParallelIndex{
			IndexNumber: pointer.Int64(parallelIndex),
		},
		Status: execution.TaskStatus{
			State:  state,
			Result: result,
		},
	}
	if state == execution.TaskTerminated {
		task.FinishTimestamp = testutils.Mkmtimep(finishTime)
	}
	return task
}
