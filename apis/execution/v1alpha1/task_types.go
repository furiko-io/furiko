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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskSpec describes a single task in the Job.
type TaskSpec struct {
	// Defines the template to create the task.
	Template TaskTemplate `json:"template"`

	// Optional duration in seconds to wait before terminating the task if it is
	// still pending. This field is useful to prevent jobs from being stuck forever
	// if the Job has a deadline to start running by. If not set, it will be set to
	// the DefaultPendingTimeoutSeconds configuration value in the controller. To
	// disable pending timeout, set this to 0.
	//
	// +optional
	PendingTimeoutSeconds *int64 `json:"pendingTimeoutSeconds,omitempty"`

	// ForbidForceDeletion, if true, means that tasks are not allowed to be
	// force deleted. If the node is unresponsive, it may be possible that the task
	// cannot be killed by normal graceful deletion. The controller may choose to
	// force delete the task, which would ignore the final state of the task since
	// the node is unable to return whether the task is actually still alive.
	//
	// As such, if not set to true, the Forbid ConcurrencyPolicy may in some cases
	// be violated. Setting this to true would prevent this from happening, but the
	// Job may remain in Killing indefinitely until the node recovers.
	//
	// +optional
	ForbidForceDeletion bool `json:"forbidForceDeletion,omitempty"`
}

// TaskTemplate defines how to create a single task for this Job. Exactly one
// field must be specified.
type TaskTemplate struct {
	// Describes how to create tasks as Pods.
	// +optional
	Pod *PodTemplateSpec `json:"pod,omitempty"`
}

// PodTemplateSpec describes the data a Pod should have when created from a template.
type PodTemplateSpec struct {
	// Standard object's metadata that will be added to Pod. More info:
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	//
	// +kubebuilder:validation:XPreserveUnknownFields
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the pod. API docs:
	// https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podspec-v1-core
	//
	// Supports context variable substitution in the following fields for containers
	// and initContainers: image, command, args, env.value
	//
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	// +mapType=atomic
	// +optional
	Spec corev1.PodSpec `json:"spec,omitempty"`
}

// ConvertToCoreSpec converts a PodTemplateSpec into the equivalent
// v1.PodTemplateSpec. This method is needed because we use a custom
// PodTemplateSpec type for the purposes of custom CRD documentation generation.
func (p *PodTemplateSpec) ConvertToCoreSpec() *corev1.PodTemplateSpec {
	return &corev1.PodTemplateSpec{
		ObjectMeta: p.ObjectMeta,
		Spec:       p.Spec,
	}
}
