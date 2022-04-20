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

package job_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
)

func TestGenerateTaskRefs(t *testing.T) {
	tests := []struct {
		name     string
		existing []execution.TaskRef
		tasks    []tasks.Task
		want     []execution.TaskRef
	}{
		{
			name:     "no tasks",
			existing: nil,
			tasks:    nil,
			want:     nil,
		},
		{
			name:     "add new task",
			existing: nil,
			tasks: []tasks.Task{
				&stubTask{
					taskRef: execution.TaskRef{
						Name:              "task1",
						CreationTimestamp: createTime,
					},
				},
			},
			want: []execution.TaskRef{
				{
					Name:              "task1",
					CreationTimestamp: createTime,
				},
			},
		},
		{
			name: "update existing task",
			existing: []execution.TaskRef{
				{
					Name:              "task1",
					CreationTimestamp: createTime,
				},
			},
			tasks: []tasks.Task{
				&stubTask{
					taskRef: execution.TaskRef{
						Name:              "task1",
						CreationTimestamp: createTime,
						FinishTimestamp:   &finishTime,
						Status: execution.TaskStatus{
							State:  execution.TaskSuccess,
							Result: job.GetResultPtr(execution.JobResultSuccess),
						},
					},
				},
			},
			want: []execution.TaskRef{
				{
					Name:              "task1",
					CreationTimestamp: createTime,
					FinishTimestamp:   &finishTime,
					Status: execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					},
					DeletedStatus: &execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					},
				},
			},
		},
		{
			name: "add new without existing task",
			existing: []execution.TaskRef{
				{
					Name:              "task1",
					CreationTimestamp: createTime,
					FinishTimestamp:   &finishTime,
					Status: execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					},
					DeletedStatus: &execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					},
				},
			},
			tasks: []tasks.Task{
				&stubTask{
					taskRef: execution.TaskRef{
						Name:              "task2",
						CreationTimestamp: createTime2,
					},
				},
			},
			want: []execution.TaskRef{
				{
					Name:              "task1",
					CreationTimestamp: createTime,
					FinishTimestamp:   &finishTime,
					Status: execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					},
					DeletedStatus: &execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					},
				},
				{
					Name:              "task2",
					CreationTimestamp: createTime2,
				},
			},
		},
		{
			name: "existing task transitioned from finished back to running",
			existing: []execution.TaskRef{
				{
					Name:              "task1",
					CreationTimestamp: createTime,
					RunningTimestamp:  &startTime,
					FinishTimestamp:   &finishTime,
					Status: execution.TaskStatus{
						State:  execution.TaskFailed,
						Result: job.GetResultPtr(execution.JobResultTaskFailed),
					},
					DeletedStatus: &execution.TaskStatus{
						State:  execution.TaskFailed,
						Result: job.GetResultPtr(execution.JobResultTaskFailed),
					},
				},
			},
			tasks: []tasks.Task{
				&stubTask{
					taskRef: execution.TaskRef{
						Name:              "task1",
						CreationTimestamp: createTime,
						RunningTimestamp:  &startTime,
						Status: execution.TaskStatus{
							State: execution.TaskStaging,
						},
					},
				},
			},
			want: []execution.TaskRef{
				{
					Name:              "task1",
					CreationTimestamp: createTime,
					RunningTimestamp:  &startTime,
					FinishTimestamp:   &finishTime,
					Status: execution.TaskStatus{
						State: execution.TaskStaging,
					},
					DeletedStatus: &execution.TaskStatus{
						State:  execution.TaskFailed,
						Result: job.GetResultPtr(execution.JobResultTaskFailed),
					},
				},
			},
		},
		{
			name:     "multiple TaskRefs with same CreationTimestamp",
			existing: []execution.TaskRef{},
			tasks: []tasks.Task{
				&stubTask{
					taskRef: execution.TaskRef{
						Name:              "task2",
						CreationTimestamp: createTime,
						RunningTimestamp:  &startTime,
						Status: execution.TaskStatus{
							State: execution.TaskStaging,
						},
					},
				},
				&stubTask{
					taskRef: execution.TaskRef{
						Name:              "task1",
						CreationTimestamp: createTime,
						RunningTimestamp:  &startTime,
						Status: execution.TaskStatus{
							State: execution.TaskStaging,
						},
					},
				},
			},
			want: []execution.TaskRef{
				{
					Name:              "task1",
					CreationTimestamp: createTime,
					RunningTimestamp:  &startTime,
					Status: execution.TaskStatus{
						State: execution.TaskStaging,
					},
				},
				{
					Name:              "task2",
					CreationTimestamp: createTime,
					RunningTimestamp:  &startTime,
					Status: execution.TaskStatus{
						State: execution.TaskStaging,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		cmpOpts := cmp.Options{
			cmpopts.EquateEmpty(),
		}
		t.Run(tt.name, func(t *testing.T) {
			if got := job.GenerateTaskRefs(tt.existing, tt.tasks); !cmp.Equal(got, tt.want, cmpOpts) {
				t.Errorf("GenerateTaskRefs() = not equal\ndiff: %v", cmp.Diff(tt.want, got, cmpOpts))
			}
		})
	}
}

func TestUpdateTaskRefDeletedStatusIfNotSet(t *testing.T) {
	createStatus := func(deletedStatus *execution.TaskStatus) execution.JobStatus {
		return execution.JobStatus{
			CreatedTasks: 1,
			Tasks: []execution.TaskRef{
				{
					Name:              "task1",
					CreationTimestamp: createTime,
					Status: execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					},
					DeletedStatus: deletedStatus,
				},
			},
		}
	}

	type args struct {
		rj       *execution.Job
		taskName string
		status   execution.TaskStatus
	}
	tests := []struct {
		name string
		args args
		want *execution.Job
	}{
		{
			name: "Task ref does not exist",
			args: args{
				rj: &execution.Job{
					Status: createStatus(&execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					}),
				},
				taskName: "task2",
				status: execution.TaskStatus{
					State:  execution.TaskKilled,
					Result: job.GetResultPtr(execution.JobResultKilled),
				},
			},
			want: &execution.Job{
				Status: createStatus(&execution.TaskStatus{
					State:  execution.TaskSuccess,
					Result: job.GetResultPtr(execution.JobResultSuccess),
				}),
			},
		},
		{
			name: "Don't update non-nil DeletedStatus",
			args: args{
				rj: &execution.Job{
					Status: createStatus(&execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					}),
				},
				taskName: "task1",
				status: execution.TaskStatus{
					State:  execution.TaskKilled,
					Result: job.GetResultPtr(execution.JobResultKilled),
				},
			},
			want: &execution.Job{
				Status: createStatus(&execution.TaskStatus{
					State:  execution.TaskSuccess,
					Result: job.GetResultPtr(execution.JobResultSuccess),
				}),
			},
		},
		{
			name: "Update nil DeletedStatus",
			args: args{
				rj: &execution.Job{
					Status: createStatus(nil),
				},
				taskName: "task1",
				status: execution.TaskStatus{
					State:  execution.TaskKilled,
					Result: job.GetResultPtr(execution.JobResultKilled),
				},
			},
			want: &execution.Job{
				Status: createStatus(&execution.TaskStatus{
					State:  execution.TaskKilled,
					Result: job.GetResultPtr(execution.JobResultKilled),
				}),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			original := tt.args.rj.DeepCopy()

			got := job.UpdateTaskRefDeletedStatusIfNotSet(tt.args.rj, tt.args.taskName, tt.args.status)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("UpdateTaskRefDeletedStatus() not equal:\ndiff: %v", cmp.Diff(tt.want, got))
			}

			if !cmp.Equal(original, tt.args.rj) {
				t.Errorf("Job was mutated:\ndiff: %v", cmp.Diff(original, tt.args.rj))
			}
		})
	}
}
