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

package tasks

import (
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
)

// Substituter knows how to substitute variables.
type Substituter func(s string) string

// MakeSubstituter returns a Substituter for a job and task spec.
func MakeSubstituter(job *execution.Job, taskSpec variablecontext.TaskSpec) Substituter {
	var subMaps []map[string]string

	// Substitutions specified in the JobSpec takes the highest priority.
	if len(job.Spec.Substitutions) > 0 {
		subMaps = append(subMaps, job.Spec.Substitutions)
	}

	// Substitute job and task context variables.
	subMaps = append(subMaps, variablecontext.ContextProvider.MakeVariablesFromJob(job))
	subMaps = append(subMaps, variablecontext.ContextProvider.MakeVariablesFromTask(taskSpec))

	// Get list of all prefixes that were injected, and drop all leftover not
	// substituted matches. This is to prevent cases like "Bad substitution" when
	// trying to interpret in Bash.
	removePrefixes := variablecontext.ContextProvider.GetAllPrefixes()
	removePrefixes = append(removePrefixes, "option.")

	// Return substitution function.
	return func(s string) string {
		return options.SubstituteVariableMaps(s, subMaps, removePrefixes)
	}
}
