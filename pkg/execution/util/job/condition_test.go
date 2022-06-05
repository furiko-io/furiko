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

// nolint: dupl
package job_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

func TestGetCondition(t *testing.T) {
	timeNow := time.Now()
	ktime.Clock = clock.NewFakeClock(timeNow)

	type args struct {
		rj         *execution.Job
		tasks      []tasks.Task
		notStarted bool
	}
	tests := []struct {
		name string
		args args
		want execution.JobCondition
	}{
		{
			name: "Not started",
			args: args{
				rj:         &execution.Job{},
				tasks:      []tasks.Task{},
				notStarted: true,
			},
			want: execution.JobCondition{
				Queueing: &execution.JobConditionQueueing{},
			},
		},
		{
			name: "ConcurrencyPolicyEnqueue",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						StartPolicy: &execution.StartPolicySpec{
							ConcurrencyPolicy: execution.ConcurrencyPolicyEnqueue,
						},
					},
				},
				tasks:      []tasks.Task{},
				notStarted: true,
			},
			want: execution.JobCondition{
				Queueing: &execution.JobConditionQueueing{
					Reason:  "Queued",
					Message: "Job is queued and may be pending other concurrent jobs to be finished",
				},
			},
		},
		{
			name: "ConcurrencyPolicyForbid",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						StartPolicy: &execution.StartPolicySpec{
							ConcurrencyPolicy: execution.ConcurrencyPolicyForbid,
						},
					},
				},
				tasks:      []tasks.Task{},
				notStarted: true,
			},
			want: execution.JobCondition{
				Queueing: &execution.JobConditionQueueing{},
			},
		},
		{
			name: "ConcurrencyPolicyAllow",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						StartPolicy: &execution.StartPolicySpec{
							ConcurrencyPolicy: execution.ConcurrencyPolicyAllow,
						},
					},
				},
				tasks:      []tasks.Task{},
				notStarted: true,
			},
			want: execution.JobCondition{
				Queueing: &execution.JobConditionQueueing{},
			},
		},
		{
			name: "Start later at a specific time",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						StartPolicy: &execution.StartPolicySpec{
							StartAfter: &startTime,
						},
					},
				},
				tasks:      []tasks.Task{},
				notStarted: true,
			},
			want: execution.JobCondition{
				Queueing: &execution.JobConditionQueueing{
					Reason:  "NotYetDue",
					Message: "Job is queued to start no earlier than " + startTime.String(),
				},
			},
		},
		{
			name: "Start later at a specific time with Enqueued",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						StartPolicy: &execution.StartPolicySpec{
							StartAfter:        &startTime,
							ConcurrencyPolicy: execution.ConcurrencyPolicyEnqueue,
						},
					},
				},
				tasks:      []tasks.Task{},
				notStarted: true,
			},
			want: execution.JobCondition{
				Queueing: &execution.JobConditionQueueing{
					Reason:  "NotYetDue",
					Message: "Job is queued to start no earlier than " + startTime.String(),
				},
			},
		},
		{
			name: "AdmissionError without StartTime",
			args: args{
				rj: &execution.Job{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							jobutil.LabelKeyAdmissionErrorMessage: "admission error message",
						},
					},
				},
				tasks:      []tasks.Task{},
				notStarted: true,
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					FinishTimestamp: metav1.NewTime(timeNow),
					Result:          execution.JobResultAdmissionError,
					Reason:          "AdmissionError",
					Message:         "admission error message",
				},
			},
		},
		{
			name: "No tasks created yet",
			args: args{
				rj:    &execution.Job{},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "PendingCreation",
					Message: "Waiting for 1 out of 1 task(s) to be created",
				},
			},
		},
		{
			name: "Job killed with no task created",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
					},
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					FinishTimestamp: killTime,
					Result:          execution.JobResultKilled,
				},
			},
		},
		{
			name: "Task is not yet running",
			args: args{
				rj: &execution.Job{
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "WaitingForTasks",
					Message: "Waiting for 1 task(s) to start running",
				},
			},
		},
		{
			name: "Task is running",
			args: args{
				rj: &execution.Job{
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{
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
			want: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					LatestCreationTimestamp: createTime,
					LatestRunningTimestamp:  startTime,
				},
			},
		},
		{
			name: "Task finished successfully",
			args: args{
				rj: &execution.Job{
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskSucceeded,
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					LatestCreationTimestamp: &createTime,
					LatestRunningTimestamp:  &startTime,
					FinishTimestamp:         finishTime,
					Result:                  execution.JobResultSuccess,
				},
			},
		},
		{
			name: "Task finished successfully with retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(2),
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskSucceeded,
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					LatestCreationTimestamp: &createTime,
					LatestRunningTimestamp:  &startTime,
					FinishTimestamp:         finishTime,
					Result:                  execution.JobResultSuccess,
				},
			},
		},
		{
			name: "Task failed without retry",
			args: args{
				rj: &execution.Job{
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
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
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					LatestCreationTimestamp: &createTime,
					LatestRunningTimestamp:  &startTime,
					FinishTimestamp:         finishTime,
					Result:                  execution.JobResultFailed,
				},
			},
		},
		{
			name: "Task failed with retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(2),
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								Result: execution.TaskFailed,
								State:  execution.TaskTerminated,
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "RetryBackoff",
					Message: "Waiting to retry creating 1 task(s)",
				},
			},
		},
		{
			name: "Task pending timeout without retry",
			args: args{
				rj: &execution.Job{
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:   execution.TaskTerminated,
								Result:  execution.TaskKilled,
								Reason:  "ImagePullBackOff",
								Message: "Back-off pulling image \"hello-world\"",
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					LatestCreationTimestamp: &createTime,
					LatestRunningTimestamp:  &startTime,
					FinishTimestamp:         finishTime,
					Result:                  execution.JobResultFailed,
				},
			},
		},
		{
			name: "Second task waiting",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(2),
						},
					},
					Status: createTaskRefsStatus("task1", "task2"),
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:   execution.TaskTerminated,
								Result:  execution.TaskFailed,
								Reason:  "ImagePullBackOff",
								Message: "Back-off pulling image \"hello-world\"",
							},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task2",
							CreationTimestamp: metav1.NewTime(createTime.Add(time.Minute)),
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "WaitingForTasks",
					Message: "Waiting for 1 task(s) to start running",
				},
			},
		},
		{
			name: "Second task running",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(2),
						},
					},
					Status: createTaskRefsStatus("task1", "task2"),
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:   execution.TaskTerminated,
								Result:  execution.TaskFailed,
								Reason:  "ImagePullBackOff",
								Message: "Back-off pulling image \"hello-world\"",
							},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task2",
							CreationTimestamp: metav1.NewTime(createTime.Add(time.Minute)),
							RunningTimestamp:  &startTime,
						},
					},
				},
			},
			want: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					LatestCreationTimestamp: createTime2,
					LatestRunningTimestamp:  startTime,
				},
			},
		},
		{
			name: "Deleted task without retry",
			args: args{
				rj: &execution.Job{
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					LatestCreationTimestamp: &createTime,
					FinishTimestamp:         metav1.NewTime(timeNow),
					Result:                  execution.JobResultFailed,
				},
			},
		},
		{
			name: "Deleted task with DeletedStatus",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(2),
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
									State:   execution.TaskRunning,
									Reason:  "DeadlineExceeded",
									Message: "...",
								},
								DeletedStatus: &execution.TaskStatus{
									State:   execution.TaskTerminated,
									Result:  execution.TaskKilled,
									Reason:  "DeadlineExceeded",
									Message: "...",
								},
							},
						},
					},
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					LatestCreationTimestamp: &createTime,
					FinishTimestamp:         finishTime,
					Result:                  execution.JobResultKilled,
				},
			},
		},
		{
			name: "Deleted task, but updated task ref when killed even with retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(2),
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
									State:   execution.TaskTerminated,
									Result:  execution.TaskKilled,
									Reason:  "DeadlineExceeded",
									Message: "...",
								},
							},
						},
					},
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					LatestCreationTimestamp: &createTime,
					FinishTimestamp:         finishTime,
					Result:                  execution.JobResultKilled,
				},
			},
		},
		{
			name: "Deleted task with retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(2),
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "RetryBackoff",
					Message: "Waiting to retry creating 1 task(s)",
				},
			},
		},
		{
			name: "Deleted task, but updated task ref when killed, have retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(2),
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
									State:   execution.TaskTerminated,
									Result:  execution.TaskKilled,
									Reason:  "DeadlineExceeded",
									Message: "...",
								},
							},
						},
					},
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "RetryBackoff",
					Message: "Waiting to retry creating 1 task(s)",
				},
			},
		},
		{
			name: "Deleted task with retry and kill timestamp",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplate{
							MaxAttempts: pointer.Int64(2),
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					LatestCreationTimestamp: &createTime,
					FinishTimestamp:         metav1.NewTime(timeNow),
					Result:                  execution.JobResultKilled,
				},
			},
		},
		{
			name: "Finished with AdmissionError",
			args: args{
				rj: &execution.Job{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							jobutil.LabelKeyAdmissionErrorMessage: "this is a message",
						},
					},
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					FinishTimestamp: metav1.NewTime(timeNow),
					Result:          execution.JobResultAdmissionError,
					Reason:          "AdmissionError",
					Message:         "this is a message",
				},
			},
		},
		{
			name: "Finished with AdmissionError with prior status",
			args: args{
				rj: &execution.Job{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							jobutil.LabelKeyAdmissionErrorMessage: "this is a message",
						},
					},
					Status: execution.JobStatus{
						Condition: execution.JobCondition{
							Finished: &execution.JobConditionFinished{
								FinishTimestamp: finishTime,
								Result:          execution.JobResultAdmissionError,
								Reason:          "AdmissionError",
								Message:         "this is an old message",
							},
						},
					},
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					FinishTimestamp: finishTime,
					Result:          execution.JobResultAdmissionError,
					Reason:          "AdmissionError",
					Message:         "this is a message",
				},
			},
		},
		{
			name: "Parallel job with no tasks created",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "PendingCreation",
					Message: "Waiting for 3 out of 3 task(s) to be created",
				},
			},
		},
		{
			name: "Parallel job with 1 remaining task to be created",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "PendingCreation",
					Message: "Waiting for 1 out of 3 task(s) to be created",
				},
			},
		},
		{
			name: "Parallel job with 1 running task but 1 remaining to be created",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
							RunningTimestamp:  &startTime,
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "PendingCreation",
					Message: "Waiting for 1 out of 3 task(s) to be created",
				},
			},
		},
		{
			name: "Parallel job with 1 running task and 2 waiting to start running",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
							RunningTimestamp:  &startTime,
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-2-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(2)},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "WaitingForTasks",
					Message: "Waiting for 2 task(s) to start running",
				},
			},
		},
		{
			name: "Parallel job with 3 running tasks",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
							RunningTimestamp:  &startTime,
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
							RunningTimestamp:  &startTime,
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-2-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(2)},
							RunningTimestamp:  &startTime,
						},
					},
				},
			},
			want: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					LatestCreationTimestamp: createTime,
					LatestRunningTimestamp:  startTime,
				},
			},
		},
		{
			name: "Parallel job with 1 successful task and 2 waiting to start running",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskSucceeded,
							},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-2-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(2)},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "WaitingForTasks",
					Message: "Waiting for 2 task(s) to start running",
				},
			},
		},
		{
			name: "Parallel job with 1 successful task and 2 waiting to start running using AnySuccessful",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AnySuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskSucceeded,
							},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-2-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(2)},
						},
					},
				},
			},
			want: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					LatestCreationTimestamp: createTime,
					LatestRunningTimestamp:  startTime,
					TerminatingTasks:        2,
				},
			},
		},
		{
			name: "Parallel job with 1 successful task and 2 running tasks",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskSucceeded,
							},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
							RunningTimestamp:  &startTime,
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-2-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(2)},
							RunningTimestamp:  &startTime,
						},
					},
				},
			},
			want: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					LatestCreationTimestamp: createTime,
					LatestRunningTimestamp:  startTime,
				},
			},
		},
		{
			name: "Parallel job with 1 successful task and 2 running tasks with AnySuccessful",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AnySuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
							RunningTimestamp:  &startTime,
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskSucceeded,
							},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
							RunningTimestamp:  &startTime,
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-2-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(2)},
							RunningTimestamp:  &startTime,
						},
					},
				},
			},
			want: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					LatestCreationTimestamp: createTime,
					LatestRunningTimestamp:  startTime,
					TerminatingTasks:        2,
				},
			},
		},
		{
			name: "Parallel job being killed with no tasks created",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					FinishTimestamp: killTime,
					Result:          execution.JobResultKilled,
				},
			},
		},
		{
			name: "Parallel job being killed with 2 created tasks",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "DeletingTasks",
					Message: "Waiting for 2 out of 3 task(s) to be deleted",
				},
			},
		},
		{
			name: "Parallel job being killed with 1 created task already deleted",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplate{
							Parallelism: &execution.ParallelismSpec{
								WithCount:          pointer.Int64(3),
								CompletionStrategy: execution.AllSuccessful,
							},
						},
					},
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-0-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(0)},
							FinishTimestamp:   &finishTime,
							Status: execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskKilled,
							},
							DeletedStatus: &execution.TaskStatus{
								State:  execution.TaskTerminated,
								Result: execution.TaskKilled,
							},
						},
					},
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task-1-1",
							CreationTimestamp: createTime,
							ParallelIndex:     &execution.ParallelIndex{IndexNumber: pointer.Int64(1)},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					Reason:  "DeletingTasks",
					Message: "Waiting for 1 out of 3 task(s) to be deleted",
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rj := tt.args.rj

			// Compute TaskRefs and update for backwards compatibility of tests.
			// We expect that UpdateJobTaskRefs will recompute the new Tasks given the list of tasks,
			// regardless of any previous value.
			// These will test that Tasks can still compute the correct Condition for the Job.
			newRj := jobutil.UpdateJobTaskRefs(rj, tt.args.tasks)

			// Set startTime if tasks is not nil.
			if !tt.args.notStarted {
				newRj.Status.StartTime = &startTime
			}

			// Compute condition from TaskRefs.
			got, err := jobutil.GetCondition(newRj)
			if err != nil {
				t.Errorf("GetCondition() returned error: %v", err)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("GetCondition() not equal:\ndiff: %v", cmp.Diff(tt.want, got))
			}
		})
	}
}

func createTaskRefsStatus(taskNames ...string) execution.JobStatus {
	refs := make([]execution.TaskRef, 0, len(taskNames))
	for i, name := range taskNames {
		refs = append(refs, execution.TaskRef{
			Name:              name,
			CreationTimestamp: metav1.NewTime(createTime.Add(time.Minute * time.Duration(i))),
		})
	}
	return execution.JobStatus{
		CreatedTasks: int64(len(refs)),
		Tasks:        refs,
	}
}
