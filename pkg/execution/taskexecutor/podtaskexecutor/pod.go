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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
)

// NewPod creates a new Pod object for the given Job and index.
func NewPod(rj *execution.Job, index int64) (*corev1.Pod, error) {
	var template corev1.PodTemplateSpec
	if jobTemplate := rj.Spec.Template; jobTemplate != nil {
		template = jobTemplate.Task.Template
	}

	// Generate name for pod.
	podName := GetPodIndexedName(rj.Name, index)
	taskTemplate := &tasks.TaskTemplate{
		Name:       podName,
		RetryIndex: index,
		PodSpec:    template.Spec,
	}

	// Generate pod spec.
	// TODO(irvinlim): This need to be moved out into the controller if we want to
	//  make the task executor generic.
	podSpec := variablecontext.SubstitutePodSpecForTask(rj, taskTemplate)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   rj.GetNamespace(),
			Name:        podName,
			Labels:      makeLabels(rj, index, template),
			Annotations: makeAnnotations(rj, index, template),
			Finalizers:  makeFinalizers(rj, index, template),
		},
		Spec: podSpec,
	}

	// Add OwnerReference back to Job
	controllerRef := metav1.NewControllerRef(rj, execution.GVKJob)
	pod.OwnerReferences = append(pod.OwnerReferences, *controllerRef)

	return pod, nil
}

func makeLabels(rj *execution.Job, index int64, template corev1.PodTemplateSpec) labels.Set {
	desiredLabels := make(labels.Set, len(template.Labels)+2)
	for k, v := range template.Labels {
		desiredLabels[k] = v
	}

	// Append additional labels.
	additionalLabels := map[string]string{
		LabelKeyJobUID:         string(rj.GetUID()),
		LabelKeyTaskRetryIndex: strconv.Itoa(int(index)),
	}
	for k, v := range additionalLabels {
		desiredLabels[k] = v
	}

	return desiredLabels
}

func makeAnnotations(_ *execution.Job, _ int64, template corev1.PodTemplateSpec) labels.Set {
	desiredAnnotations := make(labels.Set, len(template.Annotations))
	for k, v := range template.Annotations {
		desiredAnnotations[k] = v
	}
	return desiredAnnotations
}

func makeFinalizers(_ *execution.Job, _ int64, template corev1.PodTemplateSpec) []string {
	desiredFinalizers := make([]string, 0, len(template.Finalizers))
	desiredFinalizers = append(desiredFinalizers, template.Finalizers...)
	return desiredFinalizers
}
