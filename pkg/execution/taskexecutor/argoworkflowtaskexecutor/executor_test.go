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

package argoworkflowtaskexecutor_test

import (
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

const (
	jobUID       = "922cef24-a2b9-44e5-91e6-bab964401ee7"
	jobNamespace = "test"
)

var (
	fakeJob = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-sample-job",
			Namespace: jobNamespace,
			UID:       jobUID,
		},
		Spec: execution.JobSpec{
			Template: &execution.JobTemplate{
				TaskTemplate: execution.TaskTemplate{
					ArgoWorkflow: &execution.ArgoWorkflowTemplateSpec{
						Spec: v1alpha1.WorkflowSpec{
							Templates: []v1alpha1.Template{
								{
									Name: "main",
									Container: &corev1.Container{
										Name:    "main-container",
										Image:   "alpine",
										Command: []string{"true"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
)
