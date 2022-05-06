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

package mutation_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
	"github.com/furiko-io/furiko/pkg/execution/mutation"
)

func TestNewJobPatcher(t *testing.T) {
	tests := []struct {
		name         string
		operation    admissionv1.Operation
		oldRj        *v1alpha1.Job
		rj           *v1alpha1.Job
		want         *v1alpha1.Job
		wantErrors   string
		wantWarnings []string
	}{
		{
			name:      "basic mutation for create with template",
			operation: admissionv1.Create,
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					Template: &v1alpha1.JobTemplate{
						TaskTemplate: v1alpha1.TaskTemplate{
							Pod: &podTemplateSpecBare,
						},
					},
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithFinalizer,
				Spec: v1alpha1.JobSpec{
					Type: v1alpha1.JobTypeAdhoc,
					Template: &v1alpha1.JobTemplate{
						TaskTemplate: v1alpha1.TaskTemplate{
							Pod: &v1alpha1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers:    podTemplateSpecBare.Spec.Containers,
									RestartPolicy: corev1.RestartPolicyNever,
								},
							},
						},
						MaxAttempts: pointer.Int64(1),
					},
					TTLSecondsAfterFinished: config.DefaultJobExecutionConfig.DefaultTTLSecondsAfterFinished,
				},
			},
		},
		{
			name:      "basic mutation for create with ConfigName",
			operation: admissionv1.Create,
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					ConfigName: objectMetaJobConfig.Name,
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Type: v1alpha1.JobTypeAdhoc,
					Template: &v1alpha1.JobTemplate{
						TaskTemplate: jobTemplateSpecBasic.Spec.TaskTemplate,
						MaxAttempts:  pointer.Int64(1),
					},
					StartPolicy:             &startPolicyBasic,
					TTLSecondsAfterFinished: config.DefaultJobExecutionConfig.DefaultTTLSecondsAfterFinished,
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrlContext := setupContext(t, nil, []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
					},
				},
			})
			patcher := mutation.NewJobPatcher(ctrlContext)
			newRj := tt.rj.DeepCopy()
			resp := patcher.Patch(tt.operation, tt.oldRj, newRj)

			if err := checkResult(resp, tt.wantErrors, tt.wantWarnings); err != "" {
				t.Errorf("MutateJob() %v", err)
			}
			if tt.wantErrors != "" {
				return
			}

			opts := []cmp.Option{cmpopts.EquateEmpty()}
			if tt.want == nil {
				if !cmp.Equal(newRj, tt.rj, opts...) {
					t.Errorf("Patch() expected no change\ndiff = %v", cmp.Diff(tt.rj, newRj, opts...))
				}
			} else if !cmp.Equal(newRj, tt.want, opts...) {
				t.Errorf("Patch() not equal\ndiff = %v", cmp.Diff(tt.want, newRj, opts...))
			}
		})
	}
}
