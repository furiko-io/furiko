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

package argoworkflowtaskexecutor

import (
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
)

// SubstituteSpec returns a WorkflowSpec after substituting context variables.
func SubstituteSpec(
	job *execution.Job,
	wfSpec v1alpha1.WorkflowSpec,
	taskSpec variablecontext.TaskSpec,
) v1alpha1.WorkflowSpec {
	spec := wfSpec.DeepCopy()
	return substituteSpec(*spec, tasks.MakeSubstituter(job, taskSpec))
}

func substituteSpec(spec v1alpha1.WorkflowSpec, sub tasks.Substituter) v1alpha1.WorkflowSpec {
	// Substitute parameters. Users should make use of Argo's workflow inputs
	// to pass arguments into their workflow. We expose this in Furiko using the
	// Parameters field, to avoid issues from double-substitution.
	// https://argoproj.github.io/argo-workflows/workflow-inputs/
	for i, parameter := range spec.Arguments.Parameters {
		spec.Arguments.Parameters[i] = substituteParameter(parameter, sub)
	}

	return spec
}

func substituteParameter(parameter v1alpha1.Parameter, sub tasks.Substituter) v1alpha1.Parameter {
	newParam := parameter.DeepCopy()

	// We only support substituting the raw "value" field here.
	if newParam.Value != nil {
		newParam.Value = v1alpha1.AnyStringPtr(sub(string(*newParam.Value)))
	}

	return *newParam
}
