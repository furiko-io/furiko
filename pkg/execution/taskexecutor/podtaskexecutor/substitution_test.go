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

package podtaskexecutor_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
)

const (
	taskName = "sample-job.1"
	jobName  = "sample-job"
)

func TestSubstitutePodSpec(t *testing.T) {
	tests := []struct {
		name      string
		variables map[string]string
		taskSpec  variablecontext.TaskSpec
		podSpec   v1.PodSpec
		want      v1.PodSpec
	}{
		{
			name: "substitute default variables",
			taskSpec: variablecontext.TaskSpec{
				Name:       taskName,
				Namespace:  jobNamespace,
				RetryIndex: 1,
			},
			podSpec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container",
						Args: []string{"echo", "Hello"},
						Env: []v1.EnvVar{
							{Name: "JOB_NAME", Value: "${job.name}"},
							{Name: "TASK_NAME", Value: "${task.name}"},
						},
					},
				},
				InitContainers: []v1.Container{
					{
						Name:    "init-container",
						Command: []string{"sleep", "5"},
						Env: []v1.EnvVar{
							{Name: "JOB_NAME", Value: "${job.name}"},
							{Name: "TASK_NAME", Value: "${task.name}"},
						},
					},
				},
			},
			want: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container",
						Args: []string{"echo", "Hello"},
						Env: []v1.EnvVar{
							{Name: "JOB_NAME", Value: jobName},
							{Name: "TASK_NAME", Value: taskName},
						},
					},
				},
				InitContainers: []v1.Container{
					{
						Name:    "init-container",
						Command: []string{"sleep", "5"},
						Env: []v1.EnvVar{
							{Name: "JOB_NAME", Value: jobName},
							{Name: "TASK_NAME", Value: taskName},
						},
					},
				},
			},
		},
		{
			name: "substitute custom variables",
			variables: map[string]string{
				"option.arg":       "World",
				"option.image_tag": "1.2.0",
				"job.name":         "override-job-name",  // should take precedence
				"task.name":        "override-task-name", // should take precedence
			},
			taskSpec: variablecontext.TaskSpec{
				Name:       taskName,
				Namespace:  jobNamespace,
				RetryIndex: 1,
			},
			podSpec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "container",
						Image: "custom-image-tag:${option.image_tag}",
						Args:  []string{"echo", "Hello ${option.arg}"},
						Env: []v1.EnvVar{
							{Name: "JOB_NAME", Value: "${job.name}"},
							{Name: "TASK_NAME", Value: "${task.name}"},
							{Name: "ARG_NAME", Value: "${option.arg}"},
							{Name: "UNKNOWN_ARG_NAME", Value: "${option.unknown_arg}"},
						},
					},
				},
			},
			want: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "container",
						Image: "custom-image-tag:1.2.0",
						Args:  []string{"echo", "Hello World"},
						Env: []v1.EnvVar{
							{Name: "JOB_NAME", Value: "override-job-name"},
							{Name: "TASK_NAME", Value: "override-task-name"},
							{Name: "ARG_NAME", Value: "World"},
							{Name: "UNKNOWN_ARG_NAME", Value: ""},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rj := &execution.Job{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: jobNamespace,
					Name:      jobName,
				},
				Spec: execution.JobSpec{
					Substitutions: tt.variables,
				},
			}
			if got := podtaskexecutor.SubstitutePodSpec(rj, tt.podSpec, tt.taskSpec); !cmp.Equal(tt.want, got) {
				t.Errorf("SubstitutePodSpec() not equal\ndiff = %v", cmp.Diff(tt.want, got))
			}
		})
	}
}
