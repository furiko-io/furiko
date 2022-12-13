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
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
)

// NewPod returns a new Pod object for the given Job.
func NewPod(
	rj *execution.Job,
	template *corev1.PodTemplateSpec,
	index tasks.TaskIndex,
) (*corev1.Pod, error) {
	// Generate name for pod.
	podName, err := job.GenerateTaskName(rj.Name, index)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot generate pod name")
	}

	// Generate PodSpec after substitutions.
	podSpec := SubstitutePodSpec(rj, template.Spec, variablecontext.TaskSpec{
		Name:          podName,
		Namespace:     rj.GetNamespace(),
		RetryIndex:    index.Retry,
		ParallelIndex: index.Parallel,
	})

	// Label object meta for setting task labels and annotations.
	templateMeta := template.ObjectMeta.DeepCopy()
	if err := tasks.SetLabelsAnnotations(templateMeta, rj, index); err != nil {
		return nil, err
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   rj.GetNamespace(),
			Name:        podName,
			Labels:      templateMeta.Labels,
			Annotations: templateMeta.Annotations,
			Finalizers:  templateMeta.Finalizers,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(rj, execution.GVKJob),
			},
		},
		Spec: podSpec,
	}

	return pod, nil
}
