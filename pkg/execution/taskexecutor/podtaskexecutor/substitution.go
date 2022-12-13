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

package podtaskexecutor

import (
	v1 "k8s.io/api/core/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
)

// SubstitutePodSpec returns a PodSpec after substituting context variables.
func SubstitutePodSpec(
	job *execution.Job,
	podSpec v1.PodSpec,
	taskSpec variablecontext.TaskSpec,
) v1.PodSpec {
	spec := podSpec.DeepCopy()
	return substitutePodSpec(*spec, tasks.MakeSubstituter(job, taskSpec))
}

func substitutePodSpec(spec v1.PodSpec, sub tasks.Substituter) v1.PodSpec {
	// Substitute init containers.
	for i, container := range spec.InitContainers {
		spec.InitContainers[i] = substituteContainer(container, sub)
	}

	// Substitute containers.
	for i, container := range spec.Containers {
		spec.Containers[i] = substituteContainer(container, sub)
	}

	return spec
}

func substituteContainer(container v1.Container, sub tasks.Substituter) v1.Container {
	newContainer := container.DeepCopy()

	// Substitute image.
	newContainer.Image = sub(newContainer.Image)

	// Substitute env vars.
	for i, envVar := range newContainer.Env {
		newEnvVar := envVar.DeepCopy()
		newEnvVar.Value = sub(newEnvVar.Value)
		newContainer.Env[i] = *newEnvVar
	}

	// Substitute command.
	for i, cmd := range newContainer.Command {
		newContainer.Command[i] = sub(cmd)
	}

	// Substitute args.
	for i, cmd := range newContainer.Args {
		newContainer.Args[i] = sub(cmd)
	}

	return *newContainer
}
