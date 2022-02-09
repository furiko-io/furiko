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
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	jobNamespace = "test"
	jobName      = "my-sample-job"

	createTime = "2021-02-09T04:06:00Z"
	startTime  = "2021-02-09T04:06:01Z"
	now        = "2021-02-09T04:06:05Z"
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
				},
			},
		},
	}
)
