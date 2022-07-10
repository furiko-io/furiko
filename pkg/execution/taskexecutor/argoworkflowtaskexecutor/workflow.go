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
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
)

// NewWorkflow returns a new Workflow object for the given Job.
func NewWorkflow(
	rj *execution.Job,
	template *execution.ArgoWorkflowTemplateSpec,
	index tasks.TaskIndex,
) (*v1alpha1.Workflow, error) {
	name, err := job.GenerateTaskName(rj.Name, index)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot generate workflow name")
	}

	// Generate spec after substitutions.
	spec := SubstituteSpec(rj, template.Spec, variablecontext.TaskSpec{
		Name:          name,
		Namespace:     rj.GetNamespace(),
		RetryIndex:    index.Retry,
		ParallelIndex: index.Parallel,
	})

	// Label object meta for setting task labels and annotations.
	templateMeta := template.ObjectMeta.DeepCopy()
	if err := tasks.SetLabelsAnnotations(templateMeta, rj, index); err != nil {
		return nil, err
	}

	wf := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   rj.GetNamespace(),
			Name:        name,
			Labels:      templateMeta.Labels,
			Annotations: templateMeta.Annotations,
			Finalizers:  templateMeta.Finalizers,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(rj, execution.GVKJob),
			},
		},
		Spec: spec,
	}

	return wf, nil
}
