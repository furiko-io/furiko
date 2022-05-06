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
	"time"

	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
)

func TestGetNextAllowedRetry(t *testing.T) {
	tests := []struct {
		name    string
		rj      *execution.Job
		want    time.Time
		wantErr bool
	}{
		{
			name: "job has no tasks with no retry delay",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					Template: &execution.JobTemplateSpec{
						MaxAttempts: pointer.Int64(3),
					},
				},
			},
			want: time.Time{},
		},
		{
			name: "job has no tasks with retry delay",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					Template: &execution.JobTemplateSpec{
						MaxAttempts:       pointer.Int64(3),
						RetryDelaySeconds: pointer.Int64(30),
					},
				},
			},
			want: time.Time{},
		},
		{
			name: "job has no retries set",
			rj: &execution.Job{
				Spec: execution.JobSpec{},
				Status: execution.JobStatus{
					CreatedTasks: 1,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "job has some more retries with no retry delay",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					Template: &execution.JobTemplateSpec{
						MaxAttempts: pointer.Int64(3),
					},
				},
				Status: execution.JobStatus{
					CreatedTasks: 1,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			want: time.Time{},
		},
		{
			name: "job has some more retries with some delay",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					Template: &execution.JobTemplateSpec{
						MaxAttempts:       pointer.Int64(3),
						RetryDelaySeconds: pointer.Int64(30),
					},
				},
				Status: execution.JobStatus{
					CreatedTasks: 1,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			want: finishTime.Add(30 * time.Second),
		},
		{
			name: "job has some more retries with some delay, without finish timestamp set",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					Template: &execution.JobTemplateSpec{
						MaxAttempts:       pointer.Int64(3),
						RetryDelaySeconds: pointer.Int64(30),
					},
				},
				Status: execution.JobStatus{
					CreatedTasks: 1,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "job has some more retries with some delay, with kill timestamp set",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					Template: &execution.JobTemplateSpec{
						MaxAttempts:       pointer.Int64(3),
						RetryDelaySeconds: pointer.Int64(30),
					},
					KillTimestamp: &killTime,
				},
				Status: execution.JobStatus{
					CreatedTasks: 1,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "job has no more retries with no delay",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					Template: &execution.JobTemplateSpec{
						MaxAttempts:       pointer.Int64(2),
						RetryDelaySeconds: pointer.Int64(30),
					},
				},
				Status: execution.JobStatus{
					CreatedTasks: 2,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
						{
							Name:              "task2",
							CreationTimestamp: createTime2,
							FinishTimestamp:   &finishTime2,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "job has no more retries with some delay",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					Template: &execution.JobTemplateSpec{
						MaxAttempts:       pointer.Int64(2),
						RetryDelaySeconds: pointer.Int64(30),
					},
				},
				Status: execution.JobStatus{
					CreatedTasks: 2,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
						{
							Name:              "task2",
							CreationTimestamp: createTime2,
							FinishTimestamp:   &finishTime2,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := jobutil.GetNextAllowedRetry(tt.rj)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNextAllowedRetry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.want.Equal(got) {
				t.Errorf("GetNextAllowedRetry() got = %v, want %v", got, tt.want)
			}
		})
	}
}
