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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/utils/execution/job"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	jobNamespace = "test"
	jobName      = "my-sample-job"

	createTime     = "2021-02-09T04:06:00Z"
	startTime      = "2021-02-09T04:06:01Z"
	finishTime     = "2021-02-09T04:06:18Z"
	now            = "2021-02-09T04:06:05Z"
	later15m       = "2021-02-09T04:21:00Z"
	later60m       = "2021-02-09T05:06:00Z"
	finishAfterTTL = "2021-02-09T05:06:18Z"
)

var (
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

	fakeJobResult = &execution.Job{
		ObjectMeta: fakeJob.ObjectMeta,
		Spec:       fakeJob.Spec,
		Status: execution.JobStatus{
			Phase: execution.JobPending,
			Condition: execution.JobCondition{
				Waiting: &execution.JobConditionWaiting{
					CreatedAt: testutils.Mkmtimep(createTime),
				},
			},
			StartTime:    testutils.Mkmtimep(startTime),
			CreatedTasks: 1,
			Tasks: []execution.TaskRef{
				{
					Name:              fakeJob.Name + ".1",
					CreationTimestamp: testutils.Mkmtime(createTime),
					Status: execution.TaskStatus{
						State: execution.TaskStaging,
					},
					ContainerStates: []execution.TaskContainerState{},
				},
			},
		},
	}

	fakeJobFinishedResult = &execution.Job{
		ObjectMeta: fakeJobResult.ObjectMeta,
		Spec:       fakeJobResult.Spec,
		Status: execution.JobStatus{
			Phase: execution.JobSucceeded,
			Condition: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					CreatedAt:  testutils.Mkmtimep(createTime),
					StartedAt:  testutils.Mkmtimep(startTime),
					FinishedAt: testutils.Mkmtime(finishTime),
					Result:     execution.JobResultSuccess,
				},
			},
			StartTime:    testutils.Mkmtimep(startTime),
			CreatedTasks: 1,
			Tasks: []execution.TaskRef{
				{
					Name:              fakeJob.Name + ".1",
					CreationTimestamp: testutils.Mkmtime(createTime),
					RunningTimestamp:  testutils.Mkmtimep(startTime),
					FinishTimestamp:   testutils.Mkmtimep(finishTime),
					Status: execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					},
					ContainerStates: []execution.TaskContainerState{
						{ExitCode: 0},
					},
					DeletedStatus: &execution.TaskStatus{
						State:  execution.TaskSuccess,
						Result: job.GetResultPtr(execution.JobResultSuccess),
					},
				},
			},
		},
	}

	fakePod1, _ = podtaskexecutor.NewPod(fakeJob, 1)

	// Add CreationTimestamp in the result to mimic apiserver.
	fakePod1Result = func() *corev1.Pod {
		newPod := fakePod1.DeepCopy()
		newPod.CreationTimestamp = testutils.Mkmtime(createTime)
		return newPod
	}()

	// fakePod1Finished is the result for a Pod that is Succeeded.
	fakePod1Finished = func() *corev1.Pod {
		newPod := fakePod1Result.DeepCopy()
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
