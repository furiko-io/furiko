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

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	jobutil "github.com/furiko-io/furiko/pkg/utils/execution/job"
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
					FinishedAt: metav1.NewTime(timeNow),
					Result:     execution.JobResultAdmissionError,
					Reason:     "AdmissionError",
					Message:    "admission error message",
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
				Waiting: &execution.JobConditionWaiting{},
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
					FinishedAt: killTime,
					Result:     execution.JobResultKilled,
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
					CreatedAt: &createTime,
				},
			},
		},
		{
			name: "Task has pending reason",
			args: args{
				rj: &execution.Job{
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{
					&stubTask{
						taskRef: execution.TaskRef{
							Name:              "task1",
							CreationTimestamp: createTime,
							Status: execution.TaskStatus{
								Reason:  "ImagePullBackOff",
								Message: "Back-off pulling image \"hello-world\"",
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					CreatedAt: &createTime,
					Reason:    "ImagePullBackOff",
					Message:   "Back-off pulling image \"hello-world\"",
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
					CreatedAt: createTime,
					StartedAt: startTime,
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
								State:  execution.TaskSuccess,
								Result: jobutil.GetResultPtr(execution.JobResultSuccess),
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					CreatedAt:  &createTime,
					StartedAt:  &startTime,
					FinishedAt: finishTime,
					Result:     execution.JobResultSuccess,
				},
			},
		},
		{
			name: "Task finished successfully with retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxRetryAttempts: &one,
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
								State:  execution.TaskSuccess,
								Result: jobutil.GetResultPtr(execution.JobResultSuccess),
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					CreatedAt:  &createTime,
					StartedAt:  &startTime,
					FinishedAt: finishTime,
					Result:     execution.JobResultSuccess,
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
								State:  execution.TaskFailed,
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					CreatedAt:  &createTime,
					StartedAt:  &startTime,
					FinishedAt: finishTime,
					Result:     execution.JobResultTaskFailed,
				},
			},
		},
		{
			name: "Task failed with retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxRetryAttempts: &one,
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
								Result: jobutil.GetResultPtr(execution.JobResultTaskFailed),
								State:  execution.TaskFailed,
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					CreatedAt: &createTime,
					Reason:    "RetryBackoff",
					Message:   "Waiting to create new task (retrying 2 out of 2)",
				},
			},
		},
		{
			name: "Task unschedulable without retry",
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
								State:   execution.TaskPendingTimeout,
								Result:  jobutil.GetResultPtr(execution.JobResultPendingTimeout),
								Reason:  "ImagePullBackOff",
								Message: "Back-off pulling image \"hello-world\"",
							},
						},
					},
				},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					CreatedAt:  &createTime,
					StartedAt:  &startTime,
					FinishedAt: finishTime,
					Result:     execution.JobResultPendingTimeout,
					Reason:     "ImagePullBackOff",
					Message:    "Back-off pulling image \"hello-world\"",
				},
			},
		},
		{
			name: "Second task waiting",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxRetryAttempts: &one,
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
								State:   execution.TaskFailed,
								Result:  jobutil.GetResultPtr(execution.JobResultTaskFailed),
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
					CreatedAt: &createTime2,
				},
			},
		},
		{
			name: "Second task running",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxRetryAttempts: &one,
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
								State:   execution.TaskFailed,
								Result:  jobutil.GetResultPtr(execution.JobResultTaskFailed),
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
					CreatedAt: createTime2,
					StartedAt: startTime,
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
					CreatedAt:  &createTime,
					FinishedAt: metav1.NewTime(timeNow),
					Result:     execution.JobResultFinalStateUnknown,
				},
			},
		},
		{
			name: "Deleted task with DeletedStatus",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplateSpec{
							MaxRetryAttempts: &one,
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
									State:   execution.TaskKilled,
									Result:  jobutil.GetResultPtr(execution.JobResultKilled),
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
					CreatedAt:  &createTime,
					FinishedAt: finishTime,
					Result:     execution.JobResultKilled,
					Reason:     "DeadlineExceeded",
					Message:    "...",
				},
			},
		},
		{
			name: "Deleted task, but updated task ref when killed even with retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplateSpec{
							MaxRetryAttempts: &one,
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
									State:   execution.TaskKilled,
									Result:  jobutil.GetResultPtr(execution.JobResultKilled),
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
					CreatedAt:  &createTime,
					FinishedAt: finishTime,
					Result:     execution.JobResultKilled,
					Reason:     "DeadlineExceeded",
					Message:    "...",
				},
			},
		},
		{
			name: "Deleted task, but updated task ref when unschedulable",
			args: args{
				rj: &execution.Job{
					Status: execution.JobStatus{
						CreatedTasks: 1,
						Tasks: []execution.TaskRef{
							{
								Name:              "task1",
								CreationTimestamp: createTime,
								FinishTimestamp:   &finishTime,
								Status: execution.TaskStatus{
									State:   execution.TaskKilled,
									Result:  jobutil.GetResultPtr(execution.JobResultPendingTimeout),
									Reason:  "DeadlineExceeded",
									Message: "...",
								},
								DeletedStatus: &execution.TaskStatus{
									State:   execution.TaskKilled,
									Result:  jobutil.GetResultPtr(execution.JobResultPendingTimeout),
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
					CreatedAt:  &createTime,
					FinishedAt: finishTime,
					Result:     execution.JobResultPendingTimeout,
					Reason:     "DeadlineExceeded",
					Message:    "...",
				},
			},
		},
		{
			name: "Deleted task with retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxRetryAttempts: &one,
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					CreatedAt: &createTime,
					Reason:    "RetryBackoff",
					Message:   "Waiting to create new task (retrying 2 out of 2)",
				},
			},
		},
		{
			name: "Deleted task, but updated task ref when killed, have retry",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						Template: &execution.JobTemplateSpec{
							MaxRetryAttempts: &one,
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
									State:   execution.TaskKilled,
									Result:  jobutil.GetResultPtr(execution.JobResultKilled),
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
					CreatedAt: &createTime,
					Reason:    "RetryBackoff",
					Message:   "Waiting to create new task (retrying 2 out of 2)",
				},
			},
		},
		{
			name: "Deleted task with retry and kill timestamp",
			args: args{
				rj: &execution.Job{
					Spec: execution.JobSpec{
						KillTimestamp: &killTime,
						Template: &execution.JobTemplateSpec{
							MaxRetryAttempts: &one,
						},
					},
					Status: createTaskRefsStatus("task1"),
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					CreatedAt:  &createTime,
					FinishedAt: metav1.NewTime(timeNow),
					Result:     execution.JobResultFinalStateUnknown,
				},
			},
		},
		{
			name: "Finished with CannotCreateTaskError",
			args: args{
				rj: &execution.Job{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							jobutil.LabelKeyAdmissionErrorMessage: "this is a message",
						},
					},
					Spec: execution.JobSpec{},
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					FinishedAt: metav1.NewTime(timeNow),
					Result:     execution.JobResultAdmissionError,
					Reason:     "AdmissionError",
					Message:    "this is a message",
				},
			},
		},
		{
			name: "Finished with CannotCreateTaskError with prior status",
			args: args{
				rj: &execution.Job{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							jobutil.LabelKeyAdmissionErrorMessage: "this is a message",
						},
					},
					Spec: execution.JobSpec{},
					Status: execution.JobStatus{
						Condition: execution.JobCondition{
							Finished: &execution.JobConditionFinished{
								FinishedAt: finishTime,
								Result:     execution.JobResultAdmissionError,
								Reason:     "AdmissionError",
								Message:    "this is an old message",
							},
						},
					},
				},
				tasks: []tasks.Task{},
			},
			want: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					FinishedAt: finishTime,
					Result:     execution.JobResultAdmissionError,
					Reason:     "AdmissionError",
					Message:    "this is a message",
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
			if got := jobutil.GetCondition(newRj); !cmp.Equal(got, tt.want) {
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
