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
	"github.com/furiko-io/furiko/pkg/utils/execution/job"
)

func TestGetPhase(t *testing.T) {
	tests := []struct {
		name string
		rj   *execution.Job
		want execution.JobPhase
	}{
		{
			name: "Queued",
			rj:   &execution.Job{},
			want: execution.JobQueued,
		},
		{
			name: "Starting",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Waiting: &execution.JobConditionWaiting{
							CreatedAt: &createTime,
						},
					},
				},
			},
			want: execution.JobStarting,
		},
		{
			name: "Pending",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Waiting: &execution.JobConditionWaiting{
							CreatedAt: &createTime,
						},
					},
					CreatedTasks: 1,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
						},
					},
				},
			},
			want: execution.JobPending,
		},
		{
			name: "Running",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Running: &execution.JobConditionRunning{
							CreatedAt: createTime,
							StartedAt: startTime,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobRunning,
		},
		{
			name: "Succeeded",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							CreatedAt:  &createTime,
							StartedAt:  &startTime,
							FinishedAt: finishTime,
							Result:     execution.JobResultSuccess,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobSucceeded,
		},
		{
			name: "FinishedUnknown",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							CreatedAt:  &createTime,
							StartedAt:  &startTime,
							FinishedAt: finishTime,
							Result:     execution.JobResultFinalStateUnknown,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobFinishedUnknown,
		},
		{
			name: "FinishedUnknown with unhandled result",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							CreatedAt:  &createTime,
							StartedAt:  &startTime,
							FinishedAt: finishTime,
							Result:     "",
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobFinishedUnknown,
		},
		{
			name: "RetryBackoff",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Waiting: &execution.JobConditionWaiting{
							Reason: "RetryBackoff",
						},
					},
					CreatedTasks: 1,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: job.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			want: execution.JobRetryBackoff,
		},
		{
			name: "Retrying",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Waiting: &execution.JobConditionWaiting{
							CreatedAt: &createTime,
						},
					},
					CreatedTasks: 2,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskFailed,
								Result: job.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
						{
							Name:              "task1",
							CreationTimestamp: createTime2,
						},
					},
				},
			},
			want: execution.JobRetrying,
		},
		{
			name: "RetryLimitExceeded",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							CreatedAt:  &createTime,
							StartedAt:  &startTime,
							FinishedAt: finishTime,
							Result:     execution.JobResultTaskFailed,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobRetryLimitExceeded,
		},
		{
			name: "PendingTimeout",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							CreatedAt:  &createTime,
							FinishedAt: finishTime,
							Result:     execution.JobResultPendingTimeout,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobPendingTimeout,
		},
		{
			name: "DeadlineExceeded",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							CreatedAt:  &createTime,
							StartedAt:  &startTime,
							FinishedAt: finishTime,
							Result:     execution.JobResultDeadlineExceeded,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobDeadlineExceeded,
		},
		{
			name: "AdmissionError",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							CreatedAt:  &createTime,
							FinishedAt: finishTime,
							Result:     execution.JobResultAdmissionError,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobAdmissionError,
		},
		{
			name: "AdmissionError without StartTime",
			rj: &execution.Job{
				Status: execution.JobStatus{
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							CreatedAt:  &createTime,
							FinishedAt: finishTime,
							Result:     execution.JobResultAdmissionError,
						},
					},
				},
			},
			want: execution.JobAdmissionError,
		},
		{
			name: "Killing job in Waiting",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					KillTimestamp: &killTime,
				},
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Waiting: &execution.JobConditionWaiting{
							CreatedAt: &createTime,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobKilling,
		},
		{
			name: "Killing job in Running",
			rj: &execution.Job{
				Spec: execution.JobSpec{
					KillTimestamp: &killTime,
				},
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Running: &execution.JobConditionRunning{
							CreatedAt: createTime,
							StartedAt: startTime,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobKilling,
		},
		{
			name: "Killed",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							CreatedAt:  &createTime,
							StartedAt:  &startTime,
							FinishedAt: finishTime,
							Result:     execution.JobResultKilled,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobKilled,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := job.GetPhase(tt.rj); got != tt.want {
				t.Errorf("GetPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}
