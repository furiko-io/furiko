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

package validation

import (
	"sync"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/core"
	apivalidation "k8s.io/kubernetes/pkg/apis/core/validation"

	"github.com/furiko-io/furiko/pkg/core/validation"
)

var (
	scheme      = runtime.NewScheme()
	addToScheme = sync.Once{}
)

// ValidatePodTemplateSpec validates a PodTemplateSpec using Kubernetes' native
// validators and returns a validation error if any.
//
// Note that MutatingAdmissionWebhooks are always invoked before
// ValidatingAdmissionWebhooks:
// https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks
//
// Since we invoke k8s.io/kubernetes/pkg/apis/core/validation here, and it also
// makes the above assumption, we need to invoke Default() on the
// PodTemplateSpec programmatically before running any Kubernetes validators.
func ValidatePodTemplateSpec(spec *corev1.PodTemplateSpec, fieldPath *field.Path) field.ErrorList {
	podTemplateSpec := spec.DeepCopy()

	// Perform initial setup of scheme for conversions.
	var err error
	addToScheme.Do(func() {
		err = corev1.SchemeBuilder.AddToScheme(scheme)
	})
	if err != nil {
		return validation.ToInternalErrorList(fieldPath, errors.Wrapf(err, "cannot add to scheme"))
	}

	// Wrap into a new PodTemplate and perform defaulting.
	podTemplate := &corev1.PodTemplate{
		Template: *podTemplateSpec,
	}
	scheme.Default(podTemplate)

	// Use the defaulted PodTemplateSpec's template, and convert from
	// corev1.PodTemplateSpec to core.PodTemplateSpec.
	var corePodTemplateSpec core.PodTemplateSpec
	if err := scheme.Convert(&podTemplate.Template, &corePodTemplateSpec, &conversion.Meta{}); err != nil {
		return validation.ToInternalErrorList(fieldPath, errors.Wrapf(err, "conversion error"))
	}

	// Validate core.PodTemplateSpec using default Kubernetes validators.
	return apivalidation.ValidatePodTemplateSpec(
		&corePodTemplateSpec,
		fieldPath,
		apivalidation.PodValidationOptions{},
	)
}
