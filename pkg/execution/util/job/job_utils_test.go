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

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	jobtasks "github.com/furiko-io/furiko/pkg/execution/tasks"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
)

func TestAllowedToCreateNewTask(t *testing.T) {
	type args struct {
		rj    *execution.Job
		tasks []jobtasks.Task
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "No tasks created yet",
			args: args{
				rj:    &execution.Job{},
				tasks: []jobtasks.Task{},
			},
			want: true,
		},
		{
			name: "Task is not yet running",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxAttempts: &two,
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []jobtasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Task is running",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxAttempts: &two,
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []jobtasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							Status: execution.TaskStatus{
								State: execution.TaskRunning,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Task finished successfully",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxAttempts: &two,
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []jobtasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskSuccess,
								Result: jobutil.GetResultPtr(execution.JobResultSuccess),
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Task failed",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxAttempts: &two,
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []jobtasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Task failed without retry",
			args: args{
				rj: &execution.Job{
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []jobtasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Task failed but killing",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplateSpec{
							MaxAttempts: &two,
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []jobtasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Tasks failed then succeeded",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxAttempts: &three,
						},
					},
					Status: createTaskRefsStatus("task1", "task2"),
				},
				tasks: []jobtasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskSuccess,
								Result: jobutil.GetResultPtr(execution.JobResultSuccess),
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rj := tt.args.rj
			newRj := jobutil.UpdateJobTaskRefs(rj, tt.args.tasks)
			if got := jobutil.AllowedToCreateNewTask(newRj); got != tt.want {
				t.Errorf("AllowedToCreateNewTask() = %v, want %v", got, tt.want)
			}
		})
	}
}
