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

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/parallel"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

func TestGetParallelStatus(t *testing.T) {
	type args struct {
		job   *execution.Job
		tasks []execution.TaskRef
	}
	tests := []struct {
		name    string
		args    args
		want    execution.ParallelStatus
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "No tasks created",
			args: args{
				job:   &execution.Job{},
				tasks: []execution.TaskRef{},
			},
			want: execution.ParallelStatus{
				Indexes: []execution.ParallelIndexStatus{
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(0),
						},
						Hash:  makeHash(0),
						State: execution.IndexNotCreated,
					},
				},
			},
		},
		{
			name: "Task without parallel index",
			args: args{
				job: &execution.Job{},
				tasks: []execution.TaskRef{
					{
						Name:              "task1",
						CreationTimestamp: testutils.Mkmtime(createTime),
						RetryIndex:        0,
						ParallelIndex:     nil,
						Status: execution.TaskStatus{
							State: execution.TaskStarting,
						},
					},
				},
			},
			want: execution.ParallelStatus{
				Indexes: []execution.ParallelIndexStatus{
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(0),
						},
						Hash:         makeHash(0),
						CreatedTasks: 1,
						State:        execution.IndexStarting,
					},
				},
			},
		},
		{
			name: "TaskStarting",
			args: args{
				job: &execution.Job{},
				tasks: []execution.TaskRef{
					makeTaskStarting("task1", 0),
				},
			},
			want: execution.ParallelStatus{
				Indexes: []execution.ParallelIndexStatus{
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(0),
						},
						Hash:         makeHash(0),
						CreatedTasks: 1,
						State:        execution.IndexStarting,
					},
				},
			},
		},
		{
			name: "TaskRunning",
			args: args{
				job: &execution.Job{},
				tasks: []execution.TaskRef{
					makeTaskRunning("task1", 0),
				},
			},
			want: execution.ParallelStatus{
				Indexes: []execution.ParallelIndexStatus{
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(0),
						},
						Hash:         makeHash(0),
						CreatedTasks: 1,
						State:        execution.IndexRunning,
					},
				},
			},
		},
		{
			name: "TaskSucceeded",
			args: args{
				job: &execution.Job{},
				tasks: []execution.TaskRef{
					makeTaskSucceeded("task1", 0),
				},
			},
			want: execution.ParallelStatus{
				Indexes: []execution.ParallelIndexStatus{
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(0),
						},
						Hash:         makeHash(0),
						CreatedTasks: 1,
						State:        execution.IndexTerminated,
						Result:       execution.TaskSucceeded,
					},
				},
				ParallelStatusSummary: execution.ParallelStatusSummary{
					Complete:   true,
					Successful: pointer.Bool(true),
				},
			},
		},
		{
			name: "TaskRunning + TaskNotCreated",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount: pointer.Int64(2),
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskRunning("task1", 0),
				},
			},
			want: execution.ParallelStatus{
				Indexes: []execution.ParallelIndexStatus{
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(0),
						},
						Hash:         makeHash(0),
						CreatedTasks: 1,
						State:        execution.IndexRunning,
					},
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(1),
						},
						Hash:  makeHash(1),
						State: execution.IndexNotCreated,
					},
				},
			},
		},
		{
			name: "TaskSucceeded + TaskNotCreated",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount: pointer.Int64(2),
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskSucceeded("task1", 0),
				},
			},
			want: execution.ParallelStatus{
				Indexes: []execution.ParallelIndexStatus{
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(0),
						},
						Hash:         makeHash(0),
						CreatedTasks: 1,
						State:        execution.IndexTerminated,
						Result:       execution.TaskSucceeded,
					},
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(1),
						},
						Hash:  makeHash(1),
						State: execution.IndexNotCreated,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parallel.GetParallelStatus(tt.args.job, tt.args.tasks)
			if testutils.WantError(t, tt.wantErr, err,
				fmt.Sprintf("GetParallelStatus(%v, %v)", tt.args.job, tt.args.tasks)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetParallelStatus(%v, %v)", tt.args.job, tt.args.tasks)
		})
	}
}

func TestGetParallelStatusCounters(t *testing.T) {
	type args struct {
		job   *execution.Job
		tasks []execution.TaskRef
	}
	tests := []struct {
		name    string
		args    args
		want    execution.ParallelStatusCounters
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "No tasks created",
			args: args{
				job:   &execution.Job{},
				tasks: []execution.TaskRef{},
			},
			want: execution.ParallelStatusCounters{
				Terminated: 1,
			},
		},
		{
			name: "Task without parallel index",
			args: args{
				job: &execution.Job{},
				tasks: []execution.TaskRef{
					{
						Name:              "task1",
						CreationTimestamp: testutils.Mkmtime(createTime),
						RetryIndex:        0,
						ParallelIndex:     nil,
						Status: execution.TaskStatus{
							State: execution.TaskStarting,
						},
					},
				},
			},
			want: execution.ParallelStatusCounters{
				Created:  1,
				Starting: 1,
			},
		},
		{
			name: "TaskStarting",
			args: args{
				job: &execution.Job{},
				tasks: []execution.TaskRef{
					makeTaskStarting("task1", 0),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:  1,
				Starting: 1,
			},
		},
		{
			name: "TaskRunning",
			args: args{
				job: &execution.Job{},
				tasks: []execution.TaskRef{
					makeTaskRunning("task1", 0),
				},
			},
			want: execution.ParallelStatusCounters{
				Created: 1,
				Running: 1,
			},
		},
		{
			name: "TaskSucceeded",
			args: args{
				job: &execution.Job{},
				tasks: []execution.TaskRef{
					makeTaskSucceeded("task1", 0),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:    1,
				Terminated: 1,
				Succeeded:  1,
			},
		},
		{
			name: "TaskRunning + TaskNotCreated",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount: pointer.Int64(2),
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskRunning("task1", 0),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:    1,
				Running:    1,
				Terminated: 1,
			},
		},
		{
			name: "TaskSucceeded + TaskNotCreated",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount: pointer.Int64(2),
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskSucceeded("task1", 0),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:    1,
				Terminated: 2,
				Succeeded:  1,
			},
		},
		{
			name: "TaskSucceeded + TaskNotCreated, AnySuccessful",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(2),
								CompletionStrategy: execution.AnySuccessful,
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskSucceeded("task1", 0),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:    1,
				Terminated: 2,
				Succeeded:  1,
			},
		},
		{
			name: "TaskRunning + TaskStarting",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount: pointer.Int64(2),
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskRunning("task1", 0),
					makeTaskStarting("task2", 1),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:  2,
				Starting: 1,
				Running:  1,
			},
		},
		{
			name: "TaskSucceeded + TaskStarting",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount: pointer.Int64(2),
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskSucceeded("task1", 0),
					makeTaskStarting("task2", 1),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:    2,
				Starting:   1,
				Terminated: 1,
				Succeeded:  1,
			},
		},
		{
			name: "TaskSucceeded + TaskStarting, AnySuccessful",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(2),
								CompletionStrategy: execution.AnySuccessful,
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskSucceeded("task1", 0),
					makeTaskStarting("task2", 1),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:    2,
				Starting:   1,
				Terminated: 1,
				Succeeded:  1,
			},
		},
		{
			name: "TaskFailed + TaskStarting, AllSuccessful",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(2),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskFailed("task1", 0),
					makeTaskStarting("task2", 1),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:    2,
				Starting:   1,
				Terminated: 1,
				Failed:     1,
			},
		},
		{
			name: "TaskFailed + TaskStarting, AnySuccessful",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(2),
								CompletionStrategy: execution.AnySuccessful,
							},
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskFailed("task1", 0),
					makeTaskStarting("task2", 1),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:    2,
				Starting:   1,
				Terminated: 1,
				Failed:     1,
			},
		},
		{
			name: "TaskFailed + TaskStarting, AllSuccessful in retry backoff",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(2),
								CompletionStrategy: execution.AllSuccessful,
							},
							MaxAttempts: pointer.Int64(3),
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskFailed("task1", 0),
					makeTaskStarting("task2", 1),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:      2,
				Starting:     1,
				RetryBackoff: 1,
				Terminated:   1,
			},
		},
		{
			name: "TaskFailed + TaskStarting, AnySuccessful in retry backoff",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(2),
								CompletionStrategy: execution.AnySuccessful,
							},
							MaxAttempts: pointer.Int64(3),
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskFailed("task1", 0),
					makeTaskStarting("task2", 1),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:      2,
				Starting:     1,
				RetryBackoff: 1,
				Terminated:   1,
			},
		},
		{
			name: "TaskStarting + Task created second retry, AllSuccessful",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(2),
								CompletionStrategy: execution.AllSuccessful,
							},
							MaxAttempts: pointer.Int64(3),
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskFailed("task1", 0),
					makeTaskStarting("task2", 1),
					makeTaskStarting("task12", 0),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:  2,
				Starting: 2,
			},
		},
		{
			name: "TaskStarting + Task created second retry, AnySuccessful",
			args: args{
				job: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(2),
								CompletionStrategy: execution.AnySuccessful,
							},
							MaxAttempts: pointer.Int64(3),
						},
					},
				},
				tasks: []execution.TaskRef{
					makeTaskFailed("task1", 0),
					makeTaskStarting("task2", 1),
					makeTaskStarting("task12", 0),
				},
			},
			want: execution.ParallelStatusCounters{
				Created:  2,
				Starting: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := parallel.GetParallelStatus(tt.args.job, tt.args.tasks)
			if testutils.WantError(t, tt.wantErr, err,
				fmt.Sprintf("GetParallelStatus(%v, %v)", tt.args.job, tt.args.tasks)) {
				return
			}
			got := parallel.GetParallelStatusCounters(status.Indexes)
			assert.Equalf(t, tt.want, got, "GetParallelStatusCounters(%v, %v)", tt.args.job, tt.args.tasks)
		})
	}
}

func makeTaskStarting(name string, index int64) execution.TaskRef {
	return execution.TaskRef{
		Name:              name,
		CreationTimestamp: testutils.Mkmtime(createTime),
		ParallelIndex: &execution.ParallelIndex{
			IndexNumber: pointer.Int64(index),
		},
		Status: execution.TaskStatus{
			State: execution.TaskStarting,
		},
	}
}

func makeTaskRunning(name string, index int64) execution.TaskRef {
	return execution.TaskRef{
		Name:              name,
		CreationTimestamp: testutils.Mkmtime(createTime),
		RunningTimestamp:  testutils.Mkmtimep(runTime),
		ParallelIndex: &execution.ParallelIndex{
			IndexNumber: pointer.Int64(index),
		},
		Status: execution.TaskStatus{
			State: execution.TaskRunning,
		},
	}
}

func makeTaskTerminated(name string, index int64, result execution.TaskResult) execution.TaskRef {
	return execution.TaskRef{
		Name:              name,
		CreationTimestamp: testutils.Mkmtime(createTime),
		RunningTimestamp:  testutils.Mkmtimep(runTime),
		FinishTimestamp:   testutils.Mkmtimep(finishTime),
		ParallelIndex: &execution.ParallelIndex{
			IndexNumber: pointer.Int64(index),
		},
		Status: execution.TaskStatus{
			State:  execution.TaskTerminated,
			Result: result,
		},
	}
}

func makeTaskSucceeded(name string, index int64) execution.TaskRef {
	return makeTaskTerminated(name, index, execution.TaskSucceeded)
}

func makeTaskFailed(name string, index int64) execution.TaskRef {
	return makeTaskTerminated(name, index, execution.TaskFailed)
}

func makeHash(index int64) string {
	hash, err := parallel.HashIndex(execution.ParallelIndex{
		IndexNumber: pointer.Int64(index),
	})
	if err != nil {
		panic(err)
	}
	return hash
}
