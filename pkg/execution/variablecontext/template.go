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

package variablecontext

import (
	v1 "k8s.io/api/core/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
)

type subFunc func(s string) string

// SubstitutePodTemplateSpecForJob returns a PodTemplateSpec for a given Job
// after substituting context variables for the job context. Uses the Job's
// substitutions field to specify overrides for the job context.
func SubstitutePodTemplateSpecForJob(rj *execution.Job) v1.PodTemplateSpec {
	var subMaps []map[string]string
	template := rj.Spec.Template.Task.Template.DeepCopy()

	// Substitutions specified in the JobSpec takes highest priority.
	if len(rj.Spec.Substitutions) > 0 {
		subMaps = append(subMaps, rj.Spec.Substitutions)
	}

	// Substitute job context variables.
	subMaps = append(subMaps, ContextProvider.MakeVariablesFromJob(rj))

	sub := func(s string) string {
		return options.SubstituteVariableMaps(s, subMaps, []string{"job."})
	}

	// Substitute PodSpec in PodTemplateSpec.
	newPodSpec := substitutePodSpec(template.Spec, sub)
	template.Spec = newPodSpec
	return *template
}

// SubstitutePodSpecForTask returns a PodSpec from a TaskTemplate after
// substituting context variables for the task context.
func SubstitutePodSpecForTask(rj *execution.Job, template *tasks.TaskTemplate) v1.PodSpec {
	var subMaps []map[string]string
	spec := template.PodSpec.DeepCopy()

	// Substitutions specified in the JobSpec takes highest priority.
	// NOTE(irvinlim): This is duplicated from above, in case we fail to substitute
	// job context (will never happen?)
	if len(rj.Spec.Substitutions) > 0 {
		subMaps = append(subMaps, rj.Spec.Substitutions)
	}

	// Substitute task context variables.
	subMaps = append(subMaps, ContextProvider.MakeVariablesFromTask(rj, template))

	// Get list of all prefixes that were injected, and drop all leftover not
	// substituted matches. This is to prevent cases like "Bad substitution" when
	// trying to interpret in Bash. Note that we should only do this at the very
	// last substitution step.
	removePrefixes := ContextProvider.GetAllPrefixes()
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
