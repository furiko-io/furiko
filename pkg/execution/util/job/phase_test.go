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

	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
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
						Waiting: &execution.JobConditionWaiting{},
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
						Waiting: &execution.JobConditionWaiting{},
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
							LatestCreationTimestamp: createTime,
							LatestRunningTimestamp:  startTime,
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
							LatestCreationTimestamp: &createTime,
							LatestRunningTimestamp:  &startTime,
							FinishTimestamp:         finishTime,
							Result:                  execution.JobResultSuccess,
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
							LatestCreationTimestamp: &createTime,
							LatestRunningTimestamp:  &startTime,
							FinishTimestamp:         finishTime,
							Result:                  execution.JobResultFinalStateUnknown,
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
							LatestCreationTimestamp: &createTime,
							LatestRunningTimestamp:  &startTime,
							FinishTimestamp:         finishTime,
							Result:                  "",
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
								State:  execution.TaskTerminated,
								Result: execution.TaskFailed,
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
						Waiting: &execution.JobConditionWaiting{},
					},
					CreatedTasks: 2,
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskFailed,
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
			name: "Failed",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							LatestCreationTimestamp: &createTime,
							LatestRunningTimestamp:  &startTime,
							FinishTimestamp:         finishTime,
							Result:                  execution.JobResultFailed,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobFailed,
		},
		{
			name: "AdmissionError",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Finished: &execution.JobConditionFinished{
							LatestCreationTimestamp: &createTime,
							FinishTimestamp:         finishTime,
							Result:                  execution.JobResultAdmissionError,
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
							LatestCreationTimestamp: &createTime,
							FinishTimestamp:         finishTime,
							Result:                  execution.JobResultAdmissionError,
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
						Waiting: &execution.JobConditionWaiting{},
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
							LatestCreationTimestamp: createTime,
							LatestRunningTimestamp:  startTime,
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
							LatestCreationTimestamp: &createTime,
							LatestRunningTimestamp:  &startTime,
							FinishTimestamp:         finishTime,
							Result:                  execution.JobResultKilled,
						},
					},
					CreatedTasks: 1,
				},
			},
			want: execution.JobKilled,
		},
		{
			name: "Parallel Pending",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Waiting: &execution.JobConditionWaiting{
							Reason:  "WaitingForTasks",
							Message: "Waiting for 2 out of 2 task(s) to start running",
						},
					},
					CreatedTasks: 2,
					ParallelStatus: &execution.ParallelStatus{
						Indexes: []execution.ParallelIndexStatus{
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(0),
								},
								CreatedTasks: 1,
								State:        execution.IndexStarting,
							},
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(1),
								},
								CreatedTasks: 1,
								State:        execution.IndexStarting,
							},
						},
					},
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
						},
						{
							Name:              "task2",
							CreationTimestamp: createTime,
						},
					},
				},
			},
			want: execution.JobPending,
		},
		{
			name: "Parallel Running",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Running: &execution.JobConditionRunning{
							LatestCreationTimestamp: createTime,
							LatestRunningTimestamp:  startTime,
						},
					},
					CreatedTasks: 2,
					ParallelStatus: &execution.ParallelStatus{
						Indexes: []execution.ParallelIndexStatus{
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(0),
								},
								CreatedTasks: 1,
								State:        execution.IndexRunning,
							},
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(1),
								},
								CreatedTasks: 1,
								State:        execution.IndexRunning,
							},
						},
					},
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
						},
						{
							Name:              "task2",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
						},
					},
				},
			},
			want: execution.JobRunning,
		},
		{
			name: "Parallel RetryBackoff",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Waiting: &execution.JobConditionWaiting{
							Reason:  "RetryBackoff",
							Message: "Waiting to retry creating 1 task(s)",
						},
					},
					CreatedTasks: 2,
					ParallelStatus: &execution.ParallelStatus{
						Indexes: []execution.ParallelIndexStatus{
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(0),
								},
								CreatedTasks: 1,
								State:        execution.IndexRetryBackoff,
							},
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(1),
								},
								CreatedTasks: 1,
								State:        execution.IndexRunning,
							},
						},
					},
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskFailed,
							},
						},
						{
							Name:              "task2",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
						},
					},
				},
			},
			want: execution.JobRetryBackoff,
		},
		{
			name: "Parallel Retrying",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Waiting: &execution.JobConditionWaiting{
							Reason:  "WaitingForTasks",
							Message: "Waiting for 1 out of 2 task(s) to start running",
						},
					},
					CreatedTasks: 2,
					ParallelStatus: &execution.ParallelStatus{
						Indexes: []execution.ParallelIndexStatus{
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(0),
								},
								CreatedTasks: 2,
								State:        execution.IndexStarting,
							},
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(1),
								},
								CreatedTasks: 1,
								State:        execution.IndexRunning,
							},
						},
					},
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskFailed,
							},
						},
						{
							Name:              "task2",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
						},
						{
							Name:              "task12",
							CreationTimestamp: createTime,
						},
					},
				},
			},
			want: execution.JobRetrying,
		},
		{
			name: "Parallel Terminating",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Condition: execution.JobCondition{
						Running: &execution.JobConditionRunning{
							LatestCreationTimestamp: createTime,
							LatestRunningTimestamp:  startTime,
							TerminatingTasks:        1,
						},
					},
					CreatedTasks: 2,
					ParallelStatus: &execution.ParallelStatus{
						Indexes: []execution.ParallelIndexStatus{
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(0),
								},
								CreatedTasks: 2,
								State:        execution.IndexStarting,
							},
							{
								Index: execution.ParallelIndex{
									IndexNumber: pointer.Int64(1),
								},
								CreatedTasks: 1,
								State:        execution.IndexTerminated,
								Result:       execution.TaskSucceeded,
							},
						},
					},
					Tasks: []execution.TaskRef{
						{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskFailed,
							},
						},
						{
							Name:              "task2",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskSucceeded,
							},
						},
						{
							Name:              "task12",
							CreationTimestamp: createTime,
						},
					},
				},
			},
			want: execution.JobTerminating,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := job.GetPhase(tt.rj); got != tt.want {
				t.Errorf("GetPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}
