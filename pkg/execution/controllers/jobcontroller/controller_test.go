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

package jobcontroller_test

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/utils/pointer"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobcontroller"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/utils/meta"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	jobNamespace = "test"
	jobName      = "my-sample-job"

	createTime = "2021-02-09T04:06:00Z"
	startTime  = "2021-02-09T04:06:01Z"
	killTime   = "2021-02-09T04:06:10Z"
	finishTime = "2021-02-09T04:06:18Z"
	retryTime  = "2021-02-09T04:07:18Z"
	now        = "2021-02-09T04:06:05Z"
	later15m   = "2021-02-09T04:21:00Z"
	later60m   = "2021-02-09T05:06:00Z"
)

var (
	// Job that is to be created. Specify startTime initially, assume that
	// JobQueueController has already admitted the Job.
	fakeJob = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              jobName,
			Namespace:         jobNamespace,
			CreationTimestamp: testutils.Mkmtime(createTime),
			Finalizers: []string{
				executiongroup.DeleteDependentsFinalizer,
			},
		},
		Spec: execution.JobSpec{
			Type: execution.JobTypeAdhoc,
			Template: &execution.JobTemplate{
				TaskTemplate: execution.TaskTemplate{
					Pod: podTemplate,
				},
			},
		},
		Status: execution.JobStatus{
			StartTime: testutils.Mkmtimep(startTime),
		},
	}

	podTemplate = &execution.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "container",
					Image: "hello-world",
					Args: []string{
						"echo",
						"Hello world!",
					},
				},
			},
		},
	}

	// Job that was created with a fully populated status.
	fakeJobResult = generateJobStatusFromPod(fakeJob, fakePodResult)

	// Job with pod pending.
	fakeJobPending = generateJobStatusFromPod(fakeJob, fakePodPending)

	// Job with pod in killing.
	fakeJobWithPodInKilling = func() *execution.Job {
		newJob := fakeJobWithKillTimestamp.DeepCopy()
		// NOTE(irvinlim): The State is not yet updated here because the Pod is not yet reconciled
		newJob.Status.Phase = execution.JobKilling
		newJob.Status.Condition.Waiting = &execution.JobConditionWaiting{
			Reason:  "DeletingTasks",
			Message: "Waiting for 1 out of 1 task(s) to be deleted",
		}
		newJob.Status.Tasks[0].DeletedStatus = &execution.TaskStatus{
			State:  execution.TaskTerminated,
			Result: execution.TaskKilled,
		}
		return newJob
	}()

	// Job with pod in killing after update.
	fakeJobWithPodInKillingAfterUpdate = func() *execution.Job {
		newJob := fakeJobWithPodInKilling.DeepCopy()
		newJob.Status.Tasks[0].Status.State = execution.TaskKilling
		return newJob
	}()

	// Job with pod reaching pending timeout.
	fakeJobWithPodInPendingTimeout = func() *execution.Job {
		newJob := fakeJobPending.DeepCopy()
		// NOTE(irvinlim): The State is not yet updated here because the Pod is not yet reconciled
		newJob.Status.Tasks[0].DeletedStatus = &execution.TaskStatus{
			State:   execution.TaskTerminated,
			Result:  execution.TaskKilled,
			Reason:  "PendingTimeout",
			Message: "Task exceeded pending timeout of 15m0s",
		}
		return newJob
	}()

	// Job with pod reaching pending timeout after update.
	fakeJobWithPodInPendingTimeoutAfterUpdate = func() *execution.Job {
		newJob := fakeJobWithPodInPendingTimeout.DeepCopy()
		newJob.Status.Tasks[0].Status.State = execution.TaskKilling
		return newJob
	}()

	// Job with kill timestamp.
	fakeJobWithKillTimestamp = func() *execution.Job {
		newJob := fakeJobPending.DeepCopy()
		newJob.Spec.KillTimestamp = testutils.Mkmtimep(killTime)
		return newJob
	}()

	// Job with deletion timestamp.
	fakeJobWithDeletionTimestamp = func() *execution.Job {
		newJob := fakeJobPending.DeepCopy()
		newJob.DeletionTimestamp = testutils.Mkmtimep(killTime)
		return newJob
	}()

	// Job with deletion timestamp whose pods are killed.
	fakeJobWithDeletionTimestampAndKilledPods = func() *execution.Job {
		newJob := fakeJobWithDeletionTimestamp.DeepCopy()
		newJob.Status.Phase = execution.JobKilled
		newJob.Status.State = execution.JobStateFinished
		newJob.Status.Condition = execution.JobCondition{
			Finished: &execution.JobConditionFinished{
				LatestCreationTimestamp: testutils.Mkmtimep(createTime),
				FinishTimestamp:         testutils.Mkmtime(killTime),
				Result:                  execution.JobResultKilled,
			},
		}
		newJob.Status.Tasks[0].DeletedStatus = &execution.TaskStatus{
			State:   execution.TaskTerminated,
			Result:  execution.TaskKilled,
			Reason:  "JobDeleted",
			Message: "Task was killed in response to deletion of Job",
		}
		newJob.Status.Tasks[0].FinishTimestamp = testutils.Mkmtimep(killTime)
		return newJob
	}()

	// Job with deletion timestamp whose pods are killed.
	fakeJobWithDeletionTimestampAndDeletedPods = func() *execution.Job {
		newJob := fakeJobWithDeletionTimestamp.DeepCopy()
		newJob.Finalizers = meta.RemoveFinalizer(newJob.Finalizers, executiongroup.DeleteDependentsFinalizer)
		newJob.Status.Phase = execution.JobFailed
		newJob.Status.State = execution.JobStateFinished
		newJob.Status.Condition = execution.JobCondition{
			Finished: &execution.JobConditionFinished{
				LatestCreationTimestamp: testutils.Mkmtimep(createTime),
				FinishTimestamp:         testutils.Mkmtime(killTime),
				Result:                  execution.JobResultFailed,
			},
		}
		newJob.Status.Tasks[0].Status = execution.TaskStatus{
			State:   execution.TaskTerminated,
			Result:  execution.TaskKilled,
			Reason:  "JobDeleted",
			Message: "Task was killed in response to deletion of Job",
		}
		newJob.Status.Tasks[0].DeletedStatus = newJob.Status.Tasks[0].Status.DeepCopy()
		newJob.Status.Tasks[0].FinishTimestamp = testutils.Mkmtimep(killTime)
		return newJob
	}()

	// Job without pending timeout.
	fakeJobWithoutPendingTimeout = func() *execution.Job {
		newJob := fakeJobPending.DeepCopy()
		newJob.Spec.Template.TaskPendingTimeoutSeconds = pointer.Int64(0)
		return newJob
	}()

	// Job with pod being deleted.
	fakeJobPodDeleting = func() *execution.Job {
		newJob := generateJobStatusFromPod(fakeJobWithKillTimestamp, fakePodPendingDeleting)
		newJob.Status.Tasks[0].DeletedStatus = &execution.TaskStatus{
			State:   execution.TaskTerminated,
			Result:  execution.TaskKilled,
			Reason:  "Deleted",
			Message: "Task was killed via deletion",
		}
		return newJob
	}()

	// Job with pod being deleted and force deletion is not allowed.
	fakeJobPodDeletingForbidForceDeletion = func() *execution.Job {
		newJob := fakeJobPodDeleting.DeepCopy()
		newJob.Spec.Template.ForbidTaskForceDeletion = true
		return newJob
	}()

	// Job with pod being force deleted.
	fakeJobPodForceDeleting = func() *execution.Job {
		newJob := generateJobStatusFromPod(fakeJobWithKillTimestamp, fakePodPendingDeleting)
		newJob.Status.Tasks[0].DeletedStatus = &execution.TaskStatus{
			State:   execution.TaskTerminated,
			Result:  execution.TaskKilled,
			Reason:  "ForceDeleted",
			Message: "Forcefully deleted the task, container may still be running",
		}
		return newJob
	}()

	// Job with pod already deleted.
	fakeJobPodDeleted = func() *execution.Job {
		newJob := fakeJobPodDeleting.DeepCopy()
		newJob.Status.Phase = execution.JobKilled
		newJob.Status.State = execution.JobStateFinished
		newJob.Status.Condition = execution.JobCondition{
			Finished: &execution.JobConditionFinished{
				LatestCreationTimestamp: testutils.Mkmtimep(createTime),
				FinishTimestamp:         testutils.Mkmtime(now),
				Result:                  execution.JobResultKilled,
			},
		}
		newJob.Status.Tasks[0].Status = *newJob.Status.Tasks[0].DeletedStatus.DeepCopy()
		newJob.Status.Tasks[0].FinishTimestamp = testutils.Mkmtimep(now)
		return newJob
	}()

	// Job that has succeeded.
	fakeJobFinished = generateJobStatusFromPod(fakeJobResult, fakePodFinished)

	// Job that has succeeded with a custom TTLSecondsAfterFinished.
	fakeJobFinishedWithTTLAfterFinished = func() *execution.Job {
		newJob := fakeJobFinished.DeepCopy()
		newJob.Spec.TTLSecondsAfterFinished = pointer.Int64(0)
		return newJob
	}()

	// Pod that is to be created.
	fakePod0  = newPod(0, 0)
	fakePod1  = newPod(1, 0)
	fakePod2  = newPod(2, 0)
	fakePod21 = newPod(2, 1)

	// Pod that adds CreationTimestamp to mimic mutation on apiserver.
	fakePodResult = podCreated(fakePod0)

	// Pod that is in Pending state.
	fakePodPending = func() *corev1.Pod {
		newPod := fakePodResult.DeepCopy()
		newPod.Status = corev1.PodStatus{
			Phase:     corev1.PodPending,
			StartTime: testutils.Mkmtimep(startTime),
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "container",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: "cannot pull image",
						},
					},
				},
			},
		}
		return newPod
	}()

	// Pod that is in Pending state and is in the process of being deleted.
	fakePodPendingDeleting = deletePod(fakePodPending, testutils.Mkmtime(killTime))

	// Pod that is Succeeded.
	fakePodFinished = func() *corev1.Pod {
		newPod := fakePodResult.DeepCopy()
		newPod.Status = corev1.PodStatus{
			Phase:     corev1.PodSucceeded,
			StartTime: testutils.Mkmtimep(startTime),
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "container",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							StartedAt:  testutils.Mkmtime(startTime),
							FinishedAt: testutils.Mkmtime(finishTime),
						},
					},
				},
			},
		}
		return newPod
	}()

	// Job with parallelism.
	fakeJobParallel = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              jobName,
			Namespace:         jobNamespace,
			CreationTimestamp: testutils.Mkmtime(createTime),
			Finalizers: []string{
				executiongroup.DeleteDependentsFinalizer,
			},
		},
		Spec: execution.JobSpec{
			Type: execution.JobTypeAdhoc,
			Template: &execution.JobTemplate{
				TaskTemplate: execution.TaskTemplate{
					Pod: podTemplate,
				},
				Parallelism: &execution.ParallelismSpec{
					WithCount:          pointer.Int64(3),
					CompletionStrategy: execution.AllSuccessful,
				},
			},
		},
		Status: execution.JobStatus{
			StartTime: testutils.Mkmtimep(startTime),
		},
	}

	// Job that was created with a fully populated status.
	fakeJobParallelResult = generateJobStatusFromPod(
		fakeJobParallel,
		podCreated(fakePod0),
		podCreated(fakePod1),
		podCreated(fakePod2),
	)

	fakeJobParallelDelayingRetry = generateJobStatusFromPod(
		withRetryDelay(withMaxAttempts(fakeJobParallel, 2), time.Minute),
		podFinished(fakePod0),
		podFinished(fakePod1),
		podFailed(fakePod2),
	)

	fakeJobParallelRetried = generateJobStatusFromPod(
		fakeJobParallelDelayingRetry,
		podCreatedWithTime(fakePod21, testutils.Mkmtime(retryTime)),
	)

	fakeJobParallelWithDeletedFailures = func() *execution.Job {
		newJob := generateJobStatusFromPod(fakeJobParallelResult,
			podCreated(fakePod0),
			podCreated(fakePod1),
			podFailed(fakePod2),
		)
		for i := 1; i < 3; i++ {
			newJob.Status.Tasks[i].DeletedStatus = &execution.TaskStatus{
				State:  execution.TaskTerminated,
				Result: execution.TaskKilled,
			}
		}
		return newJob
	}()
)

var (
	podCreateReactor = newPodCreateReactor(testutils.Mkmtime(createTime))
)

func newPodCreateReactor(createTime metav1.Time) *ktesting.SimpleReactor {
	return &ktesting.SimpleReactor{
		Verb:     "create",
		Resource: "pods",
		Reaction: func(action ktesting.Action) (bool, runtime.Object, error) {
			createAction, ok := action.(ktesting.CreateAction)
			if !ok {
				return false, nil, fmt.Errorf("not a CreateAction: %v", action)
			}
			pod, ok := createAction.GetObject().(*corev1.Pod)
			if !ok {
				return false, nil, fmt.Errorf("not a Pod: %T", createAction.GetObject())
			}
			return true, podCreatedWithTime(pod, createTime), nil
		},
	}
}

func newPod(parallelIndex int64, retryIndex int64) *corev1.Pod {
	pod, _ := podtaskexecutor.NewPod(fakeJob, podTemplate.ConvertToCoreSpec(), tasks.TaskIndex{
		Retry: retryIndex,
		Parallel: execution.ParallelIndex{
			IndexNumber: pointer.Int64(parallelIndex),
		},
	})
	return pod
}

func podCreated(pod *corev1.Pod) *corev1.Pod {
	return podCreatedWithTime(pod, testutils.Mkmtime(createTime))
}

func podCreatedWithTime(pod *corev1.Pod, createTime metav1.Time) *corev1.Pod {
	newPod := pod.DeepCopy()
	newPod.CreationTimestamp = createTime
	return newPod
}

func podFinished(pod *corev1.Pod) *corev1.Pod {
	newPod := podCreated(pod)
	newPod.Status = corev1.PodStatus{
		Phase:     corev1.PodSucceeded,
		StartTime: testutils.Mkmtimep(startTime),
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name: "container",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						StartedAt:  testutils.Mkmtime(startTime),
						FinishedAt: testutils.Mkmtime(finishTime),
					},
				},
			},
		},
	}
	return newPod
}

func podFailed(pod *corev1.Pod) *corev1.Pod {
	newPod := podFinished(pod)
	newPod.Status.Phase = corev1.PodFailed
	return newPod
}

func withMaxAttempts(job *execution.Job, maxAttempts int64) *execution.Job {
	newJob := job.DeepCopy()
	newJob.Spec.Template.MaxAttempts = pointer.Int64(maxAttempts)
	return newJob
}

func withRetryDelay(job *execution.Job, delay time.Duration) *execution.Job {
	newJob := job.DeepCopy()
	newJob.Spec.Template.RetryDelaySeconds = pointer.Int64(int64(delay.Seconds()))
	return newJob
}

func withCompletionStrategy(job *execution.Job, strategy execution.ParallelCompletionStrategy) *execution.Job {
	newJob := job.DeepCopy()
	newJob.Spec.Template.Parallelism.CompletionStrategy = strategy
	return newJob
}

// generateJobStatusFromPod returns a new Job whose status is reconciled from
// the Pod.
//
// This method is basically identical to what is used in the controller, and is
// meant to reduce flaky tests by coupling any changes to internal logic to the
// fixtures themselves, thus making it suitable for integration tests.
func generateJobStatusFromPod(rj *execution.Job, pods ...*corev1.Pod) *execution.Job {
	podTasks := make([]tasks.Task, 0, len(pods))
	for _, pod := range pods {
		podTasks = append(podTasks, podtaskexecutor.NewPodTask(pod, nil))
	}
	newJob := job.UpdateJobTaskRefs(rj, podTasks)
	newRj, err := jobcontroller.UpdateJobStatusFromTaskRefs(newJob)
	if err != nil {
		panic(err) // panic ok for tests
	}
	return newRj
}

// deletePod returns a new Pod after setting the deletion timestamp.
func deletePod(pod *corev1.Pod, ts metav1.Time) *corev1.Pod {
	newPod := pod.DeepCopy()
	newPod.DeletionTimestamp = &ts
	return newPod
}
