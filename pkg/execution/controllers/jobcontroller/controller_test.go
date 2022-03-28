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
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobcontroller"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/utils/execution/job"
	"github.com/furiko-io/furiko/pkg/utils/k8sutils"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	jobNamespace = "test"
	jobName      = "my-sample-job"

	createTime     = "2021-02-09T04:06:00Z"
	startTime      = "2021-02-09T04:06:01Z"
	killTime       = "2021-02-09T04:06:10Z"
	finishTime     = "2021-02-09T04:06:18Z"
	now            = "2021-02-09T04:06:05Z"
	later15m       = "2021-02-09T04:21:00Z"
	later60m       = "2021-02-09T05:06:00Z"
	finishAfterTTL = "2021-02-09T05:06:18Z"
)

var (
	// Job that is to be created. Specify startTime initially, assume that
	// JobQueueController has already admitted the Job.
	fakeJob = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              jobName,
			Namespace:         jobNamespace,
			CreationTimestamp: testutils.Mkmtime(createTime),
		},
		Spec: execution.JobSpec{
			Type: execution.JobTypeAdhoc,
			Template: &execution.JobTemplateSpec{
				Task: execution.JobTaskSpec{
					Template: corev1.PodTemplateSpec{
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
					},
				},
			},
		},
		Status: execution.JobStatus{
			StartTime: testutils.Mkmtimep(startTime),
		},
	}

	// Job that was created with a fully populated status.
	fakeJobResult = generateJobStatusFromPod(fakeJob, fakePodResult)

	// Job with pod pending.
	fakeJobPending = generateJobStatusFromPod(fakeJob, fakePodPending)

	// Job with kill timestamp.
	fakeJobWithKillTimestamp = func() *execution.Job {
		newJob := fakeJobPending.DeepCopy()
		newJob.Spec.KillTimestamp = testutils.Mkmtimep(killTime)
		return newJob
	}()

	// Job without pending timeout.
	fakeJobWithoutPendingTimeout = func() *execution.Job {
		newJob := fakeJobPending.DeepCopy()
		newJob.Spec.Template.Task.PendingTimeoutSeconds = pointer.Int64(0)
		return newJob
	}()

	// Job that has succeeded.
	fakeJobFinished = generateJobStatusFromPod(fakeJobResult, fakePodFinished)

	// Pod that is to be created.
	fakePod, _ = podtaskexecutor.NewPod(fakeJob, 1)

	// Pod that adds CreationTimestamp to mimic mutation on apiserver.
	fakePodResult = func() *corev1.Pod {
		newPod := fakePod.DeepCopy()
		newPod.CreationTimestamp = testutils.Mkmtime(createTime)
		return newPod
	}()

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

	// Pod that is in Pending state and is in the process of being killed.
	fakePodTerminating = killPod(fakePodPending, testutils.Mktime(killTime))

	// Pod that is in Pending state and is in the process of being killed by pending
	// timeout.
	fakePodPendingTimeoutTerminating = func() *corev1.Pod {
		newPod := killPod(fakePodPending, testutils.Mktime(later15m))
		k8sutils.SetAnnotation(newPod, podtaskexecutor.LabelKeyKilledFromPendingTimeout, "1")
		return newPod
	}()

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
)

// generateJobStatusFromPod returns a new Job whose status is reconciled from
// the Pod.
//
// This method is basically identical to what is used in the controller, and is
// meant to reduce flaky tests by coupling any changes to internal logic to the
// fixtures themselves, thus making it suitable for integration tests.
func generateJobStatusFromPod(rj *execution.Job, pod *corev1.Pod) *execution.Job {
	newJob := job.UpdateJobTaskRefs(rj, []tasks.Task{podtaskexecutor.NewPodTask(pod, nil)})
	return jobcontroller.UpdateJobStatusFromTaskRefs(newJob)
}

// killPod returns a new Pod after setting the kill timestamp.
func killPod(pod *corev1.Pod, ts time.Time) *corev1.Pod {
	newPod := pod.DeepCopy()
	k8sutils.SetAnnotation(newPod, podtaskexecutor.LabelKeyTaskKillTimestamp, strconv.Itoa(int(ts.Unix())))
	newPod.Spec.ActiveDeadlineSeconds = pointer.Int64(int64(ts.Sub(newPod.Status.StartTime.Time).Seconds()))
	return newPod
}
