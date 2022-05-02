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
	"github.com/furiko-io/furiko/pkg/core/options"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
)

type subFunc func(s string) string

// SubstitutePodSpec returns a PodSpec after substituting context variables.
func SubstitutePodSpec(
	rj *execution.Job,
	podSpec v1.PodSpec,
	taskSpec variablecontext.TaskSpec,
) v1.PodSpec {
	var subMaps []map[string]string
	spec := podSpec.DeepCopy()

	// Substitutions specified in the JobSpec takes the highest priority.
	if len(rj.Spec.Substitutions) > 0 {
		subMaps = append(subMaps, rj.Spec.Substitutions)
	}

	// Substitute job and task context variables.
	subMaps = append(subMaps, variablecontext.ContextProvider.MakeVariablesFromJob(rj))
	subMaps = append(subMaps, variablecontext.ContextProvider.MakeVariablesFromTask(taskSpec))

	// Get list of all prefixes that were injected, and drop all leftover not
	// substituted matches. This is to prevent cases like "Bad substitution" when
	// trying to interpret in Bash.
	removePrefixes := variablecontext.ContextProvider.GetAllPrefixes()
	removePrefixes = append(removePrefixes, "option.")

	sub := func(s string) string {
		return options.SubstituteVariableMaps(s, subMaps, removePrefixes)
	}

	return substitutePodSpec(*spec, sub)
}

func substitutePodSpec(spec v1.PodSpec, sub subFunc) v1.PodSpec {
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

func substituteContainer(container v1.Container, sub subFunc) v1.Container {
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
