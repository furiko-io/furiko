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

package validation_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/furiko-io/furiko/pkg/execution/validation"
)

var (
	podTemplateSpecEmpty = corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{},
		},
	}

	podTemplateSpecBasic = corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "container",
					Image: "alpine",
					Command: []string{
						"echo",
						"Hello World",
					},
				},
			},
		},
	}
)

func TestValidatePodTemplateSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    *corev1.PodTemplateSpec
		wantErr string
	}{
		{
			name:    "empty spec",
			spec:    &podTemplateSpecEmpty,
			wantErr: "[].spec.containers: Required value",
		},
		{
			name: "basic spec",
			spec: &podTemplateSpecBasic,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			original := tt.spec.DeepCopy()
			fieldPath := field.NewPath("")
			err := validation.ValidatePodTemplateSpec(tt.spec, fieldPath).ToAggregate()
			if (err == nil) != (tt.wantErr == "") || err != nil && err.Error() != tt.wantErr {
				t.Errorf("ValidatePodTemplateSpec() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !cmp.Equal(tt.spec, original) {
				t.Errorf("ValidatePodTemplateSpec() mutated input\ndiff = %v", cmp.Diff(original, tt.spec))
			}
		})
	}
}
