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

// nolint:lll
package mutation_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/apis/execution"
	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
	"github.com/furiko-io/furiko/pkg/core/options"
	"github.com/furiko-io/furiko/pkg/execution/mutation"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/runtime/webhook"
)

const (
	cronSchedule1 = "0 5 * * *"
	cronSchedule2 = "45 17 * * *"
)

var (
	now, _         = time.Parse(time.RFC3339, "2022-03-16T08:00:00Z")
	mockNow        = metav1.NewTime(now)
	mockOldTime    = metav1.NewTime(mockNow.Add(-time.Hour))
	mockFutureTime = metav1.NewTime(mockNow.Add(time.Hour))

	objectMetaJobConfig = metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      "jobconfig-sample",
		UID:       "7172ef3a-7754-4e73-a99a-d7e8a8e9fe5b",
	}

	ownerReferences = []metav1.OwnerReference{
		{
			APIVersion:         v1alpha1.GroupVersion.String(),
			Kind:               v1alpha1.KindJobConfig,
			Name:               objectMetaJobConfig.Name,
			UID:                objectMetaJobConfig.UID,
			Controller:         pointer.Bool(true),
			BlockOwnerDeletion: pointer.Bool(true),
		},
	}

	objectMetaJob = metav1.ObjectMeta{
		Namespace: metav1.NamespaceDefault,
		Name:      "job",
	}

	objectMetaJobWithFinalizer = metav1.ObjectMeta{
		Namespace:  metav1.NamespaceDefault,
		Name:       "job",
		Finalizers: []string{execution.DeleteDependentsFinalizer},
	}

	objectMetaJobWithAllReferences = metav1.ObjectMeta{
		Namespace:  metav1.NamespaceDefault,
		Name:       "job",
		Finalizers: []string{execution.DeleteDependentsFinalizer},
		Labels: map[string]string{
			jobconfig.LabelKeyJobConfigUID: string(objectMetaJobConfig.UID),
		},
		OwnerReferences: ownerReferences,
	}

	objectMetaJobWithOptionsHash = metav1.ObjectMeta{
		Namespace:       objectMetaJobWithAllReferences.Namespace,
		Name:            objectMetaJobWithAllReferences.Name,
		Labels:          objectMetaJobWithAllReferences.Labels,
		OwnerReferences: objectMetaJobWithAllReferences.OwnerReferences,
		Finalizers:      objectMetaJobWithAllReferences.Finalizers,
		Annotations: map[string]string{
			jobconfig.AnnotationKeyOptionSpecHash: optionSpecHash,
		},
	}

	podTemplateSpecBare = v1alpha1.PodTemplateSpec{
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

	podTemplateSpecBasic = v1alpha1.PodTemplateSpec{
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
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	podTemplateSpecBasic2 = v1alpha1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "container",
					Image: "alpine",
					Command: []string{
						"echo",
						"Hello World 2",
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	jobTemplateSpecBasic = v1alpha1.JobTemplate{
		Spec: v1alpha1.JobTemplateSpec{
			Task: v1alpha1.TaskSpec{
				Template: v1alpha1.TaskTemplate{
					Pod: &podTemplateSpecBasic,
				},
			},
		},
	}

	jobTemplateSpecBasic2 = v1alpha1.JobTemplate{
		Spec: v1alpha1.JobTemplateSpec{
			Task: v1alpha1.TaskSpec{
				Template: v1alpha1.TaskTemplate{
					Pod: &podTemplateSpecBasic2,
				},
			},
		},
	}

	concurrencySpecBasic = v1alpha1.ConcurrencySpec{
		Policy: v1alpha1.ConcurrencyPolicyForbid,
	}

	startPolicyBasic = v1alpha1.StartPolicySpec{
		ConcurrencyPolicy: concurrencySpecBasic.Policy,
	}

	startPolicyOverride = v1alpha1.StartPolicySpec{
		ConcurrencyPolicy: v1alpha1.ConcurrencyPolicyAllow,
	}

	optionSpecThree = v1alpha1.OptionSpec{
		Options: []v1alpha1.Option{
			{
				Type:     v1alpha1.OptionTypeString,
				Name:     "option1",
				Required: true,
			},
			{
				Type: v1alpha1.OptionTypeString,
				Name: "option2",
				String: &v1alpha1.StringOptionConfig{
					Default: "value2",
				},
			},
			{
				Type: v1alpha1.OptionTypeString,
				Name: "option3",
				String: &v1alpha1.StringOptionConfig{
					Default: "value3",
				},
			},
		},
	}

	optionSpecHash, _ = options.HashOptionSpec(&optionSpecThree)
)

func TestMutator_MutateJobConfig(t *testing.T) {
	tests := []struct {
		name         string
		rjc          *v1alpha1.JobConfig
		want         *v1alpha1.JobConfig
		wantErrors   string
		wantWarnings []string
	}{
		{
			name: "no change expected",
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
				},
			},
		},
		{
			name: "don't need to default JobTemplate",
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: v1alpha1.JobTemplate{
						Spec: v1alpha1.JobTemplateSpec{
							Task: v1alpha1.TaskSpec{
								Template: v1alpha1.TaskTemplate{
									Pod: &podTemplateSpecBare,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "default OptionSpec",
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Option: &v1alpha1.OptionSpec{
						Options: []v1alpha1.Option{
							{
								Type: v1alpha1.OptionTypeBool,
								Name: "opt",
							},
						},
					},
				},
			},
			want: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Option: &v1alpha1.OptionSpec{
						Options: []v1alpha1.Option{
							{
								Type: v1alpha1.OptionTypeBool,
								Name: "opt",
								Bool: &v1alpha1.BoolOptionConfig{
									Format: v1alpha1.BoolOptionFormatTrueFalse,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutator := mutation.NewMutator(mock.NewContext())
			newRjc := tt.rjc.DeepCopy()
			resp := mutator.MutateJobConfig(newRjc)

			if err := checkResult(resp, tt.wantErrors, tt.wantWarnings); err != "" {
				t.Errorf("MutateJobConfig() %v", err)
			}
			if tt.wantErrors != "" {
				return
			}

			opts := []cmp.Option{cmpopts.EquateEmpty()}
			if tt.want == nil {
				if !cmp.Equal(newRjc, tt.rjc, opts...) {
					t.Errorf("MutateJobConfig() expected no change\ndiff = %v", cmp.Diff(tt.rjc, newRjc, opts...))
				}
			} else if !cmp.Equal(newRjc, tt.want, opts...) {
				t.Errorf("MutateJobConfig() not equal\ndiff = %v", cmp.Diff(tt.want, newRjc, opts...))
			}
		})
	}
}

func TestMutator_MutateCreateJobConfig(t *testing.T) {
	tests := []struct {
		name         string
		rjc          *v1alpha1.JobConfig
		want         *v1alpha1.JobConfig
		wantErrors   string
		wantWarnings []string
		setup        func()
	}{
		{
			name: "don't add NotBefore without schedule",
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
				},
			},
		},
		{
			name: "bump LastUpdatedTime with updated cron schedule",
			setup: func() {
				mutation.Clock = clock.NewFakeClock(mockNow.Time)
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule1, nil),
				},
			},
			want: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule1, &mockNow),
				},
			},
		},
		{
			name: "don't reset future LastUpdated",
			setup: func() {
				mutation.Clock = clock.NewFakeClock(mockNow.Time)
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule1, &mockFutureTime),
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutator := mutation.NewMutator(mock.NewContext())
			if tt.setup != nil {
				tt.setup()
			}

			newRjc := tt.rjc.DeepCopy()
			resp := mutator.MutateCreateJobConfig(newRjc)

			if err := checkResult(resp, tt.wantErrors, tt.wantWarnings); err != "" {
				t.Errorf("MutateJob() %v", err)
			}
			if tt.wantErrors != "" {
				return
			}

			opts := []cmp.Option{cmpopts.EquateEmpty()}
			if tt.want == nil {
				if !cmp.Equal(newRjc, tt.rjc, opts...) {
					t.Errorf("MutateCreateJobConfig() expected no change\ndiff = %v", cmp.Diff(tt.rjc, newRjc, opts...))
				}
			} else if !cmp.Equal(newRjc, tt.want, opts...) {
				t.Errorf("MutateCreateJobConfig() not equal\ndiff = %v", cmp.Diff(tt.want, newRjc, opts...))
			}
		})
	}
}

func TestMutator_MutateUpdateJobConfig(t *testing.T) {
	tests := []struct {
		name         string
		oldRjc       *v1alpha1.JobConfig
		rjc          *v1alpha1.JobConfig
		want         *v1alpha1.JobConfig
		wantErrors   string
		wantWarnings []string
		setup        func()
	}{
		{
			name: "don't add LastUpdated without schedule",
			oldRjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
				},
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic2,
				},
			},
		},
		{
			name: "don't add LastUpdated when removing schedule",
			oldRjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule1, &mockOldTime),
				},
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
				},
			},
		},
		{
			name: "no change expected",
			setup: func() {
				mutation.Clock = clock.NewFakeClock(mockNow.Time)
			},
			oldRjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule1, &mockOldTime),
				},
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule1, &mockOldTime),
				},
			},
		},
		{
			name: "no change expected without schedule",
			setup: func() {
				mutation.Clock = clock.NewFakeClock(mockNow.Time)
			},
			oldRjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
				},
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
				},
			},
		},
		{
			name: "bump LastUpdated",
			setup: func() {
				mutation.Clock = clock.NewFakeClock(mockNow.Time)
			},
			oldRjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule1, &mockOldTime),
				},
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule2, &mockOldTime),
				},
			},
			want: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule2, &mockNow),
				},
			},
		},
		{
			name: "bump LastUpdated add schedule",
			setup: func() {
				mutation.Clock = clock.NewFakeClock(mockNow.Time)
			},
			oldRjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
				},
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule2, nil),
				},
			},
			want: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule2, &mockNow),
				},
			},
		},
		{
			name: "bump LastUpdated update disabled",
			setup: func() {
				mutation.Clock = clock.NewFakeClock(mockNow.Time)
			},
			oldRjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: &v1alpha1.ScheduleSpec{
						Cron: &v1alpha1.CronSchedule{
							Expression: cronSchedule1,
							Timezone:   "Asia/Singapore",
						},
						Disabled: true,
					},
				},
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: &v1alpha1.ScheduleSpec{
						Cron: &v1alpha1.CronSchedule{
							Expression: cronSchedule1,
							Timezone:   "Asia/Singapore",
						},
					},
				},
			},
			want: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule1, &mockNow),
				},
			},
		},
		{
			name: "don't reset future LastUpdated",
			setup: func() {
				mutation.Clock = clock.NewFakeClock(mockNow.Time)
			},
			oldRjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule1, &mockFutureTime),
				},
			},
			rjc: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule2, &mockFutureTime),
				},
			},
			want: &v1alpha1.JobConfig{
				ObjectMeta: objectMetaJobConfig,
				Spec: v1alpha1.JobConfigSpec{
					Template: jobTemplateSpecBasic,
					Schedule: mkScheduleSpec(cronSchedule2, &mockFutureTime),
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutator := mutation.NewMutator(mock.NewContext())
			if tt.setup != nil {
				tt.setup()
			}

			newRjc := tt.rjc.DeepCopy()
			resp := mutator.MutateUpdateJobConfig(tt.oldRjc, newRjc)

			if err := checkResult(resp, tt.wantErrors, tt.wantWarnings); err != "" {
				t.Errorf("MutateJob() %v", err)
			}
			if tt.wantErrors != "" {
				return
			}

			opts := []cmp.Option{cmpopts.EquateEmpty()}
			if tt.want == nil {
				if !cmp.Equal(newRjc, tt.rjc, opts...) {
					t.Errorf("MutateUpdateJobConfig() expected no change\ndiff = %v", cmp.Diff(tt.rjc, newRjc, opts...))
				}
			} else if !cmp.Equal(newRjc, tt.want, opts...) {
				t.Errorf("MutateUpdateJobConfig() not equal\ndiff = %v", cmp.Diff(tt.want, newRjc, opts...))
			}
		})
	}
}

func TestMutator_MutateJob(t *testing.T) {
	tests := []struct {
		name         string
		cfgs         map[configv1alpha1.ConfigName]runtime.Object
		rj           *v1alpha1.Job
		want         *v1alpha1.Job
		wantErrors   string
		wantWarnings []string
	}{
		{
			name: "basic mutation",
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					Template: &v1alpha1.JobTemplateSpec{
						Task: v1alpha1.TaskSpec{
							Template: v1alpha1.TaskTemplate{
								Pod: &podTemplateSpecBasic,
							},
						},
					},
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					Type: v1alpha1.JobTypeAdhoc,
					Template: &v1alpha1.JobTemplateSpec{
						Task: v1alpha1.TaskSpec{
							Template: v1alpha1.TaskTemplate{
								Pod: &podTemplateSpecBasic,
							},
						},
						MaxAttempts: pointer.Int64(1),
					},
					TTLSecondsAfterFinished: config.DefaultJobExecutionConfig.DefaultTTLSecondsAfterFinished,
				},
			},
		},
		{
			name: "no change expected",
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					Type: v1alpha1.JobTypeAdhoc,
					Template: &v1alpha1.JobTemplateSpec{
						Task: v1alpha1.TaskSpec{
							Template: v1alpha1.TaskTemplate{
								Pod: &podTemplateSpecBasic,
							},
						},
						MaxAttempts: pointer.Int64(1),
					},
					TTLSecondsAfterFinished: config.DefaultJobExecutionConfig.DefaultTTLSecondsAfterFinished,
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutator := setup(t, tt.cfgs, nil)
			newRj := tt.rj.DeepCopy()
			resp := mutator.MutateJob(newRj)

			if err := checkResult(resp, tt.wantErrors, tt.wantWarnings); err != "" {
				t.Errorf("MutateJob() %v", err)
			}
			if tt.wantErrors != "" {
				return
			}

			opts := []cmp.Option{cmpopts.EquateEmpty()}
			if tt.want == nil {
				if !cmp.Equal(newRj, tt.rj, opts...) {
					t.Errorf("MutateJob() expected no change\ndiff = %v", cmp.Diff(tt.rj, newRj, opts...))
				}
			} else if !cmp.Equal(newRj, tt.want, opts...) {
				t.Errorf("MutateJob() not equal\ndiff = %v", cmp.Diff(tt.want, newRj, opts...))
			}
		})
	}
}

func TestMutator_MutateCreateJob(t *testing.T) {
	tests := []struct {
		name         string
		cfgs         map[configv1alpha1.ConfigName]runtime.Object
		rj           *v1alpha1.Job
		want         *v1alpha1.Job
		rjcs         []*v1alpha1.JobConfig
		setup        func()
		wantErrors   string
		wantWarnings []string
	}{
		{
			name: "no mutation needed",
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithFinalizer,
				Spec: v1alpha1.JobSpec{
					Template: &jobTemplateSpecBasic.Spec,
				},
			},
		},
		{
			name: "add default fields without JobConfig",
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					Template: &jobTemplateSpecBasic.Spec,
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithFinalizer,
				Spec: v1alpha1.JobSpec{
					Template: &jobTemplateSpecBasic.Spec,
				},
			},
		},
		{
			name: "no mutation needed with JobConfig",
			setup: func() {
				variablecontext.ContextProvider = &mockProvider{
					ProvideForJobConfig: func(rjc *v1alpha1.JobConfig) map[string]string {
						return map[string]string{
							"jobconfig.provider_value_1": "default_provider_value_1",
							"jobconfig.provider_value_2": "default_provider_value_2",
						}
					},
				}
			},
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
					Substitutions: map[string]string{
						"jobconfig.provider_value_1": "default_provider_value_1",
						"jobconfig.provider_value_2": "default_provider_value_2",
					},
				},
			},
		},
		{
			name: "mutate optionValues as YAML",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
						Option:      &optionSpecThree,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
					OptionValues: optionValuesYAML(map[string]interface{}{
						"option1": "newvalue1",
						"option2": "newvalue2",
					}),
					Substitutions: map[string]string{
						"option.option2": "substitutionvalue2",
					},
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithOptionsHash,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "newvalue1",
						"option2": "newvalue2",
					}),
					Substitutions: map[string]string{
						"option.option1": "newvalue1",
						"option.option2": "substitutionvalue2",
						"option.option3": "value3",
					},
				},
			},
		},
		{
			name: "mutate optionValues no JobConfig",
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template: &jobTemplateSpecBasic.Spec,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "newvalue1",
						"option2": "newvalue2",
					}),
				},
			},
			wantErrors: `metadata.ownerReferences[0]: Not found: "jobconfig-sample"`,
		},
		{
			name: "optionValues will be ignored no reference",
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithFinalizer,
				Spec: v1alpha1.JobSpec{
					Template: &jobTemplateSpecBasic.Spec,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "newvalue1",
						"option2": "newvalue2",
					}),
				},
			},
			wantWarnings: []string{`optionValues will be ignored without a JobConfig: {"option1":"newvalue1","option2":"newvalue2"}`},
		},
		{
			name: "invalid YAML/JSON in optionValues",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template:     &jobTemplateSpecBasic.Spec,
					OptionValues: `asd`,
				},
			},
			wantErrors: `spec.optionValues: Invalid value: "asd": cannot unmarshal as json or yaml`,
		},
		{
			name: "required option and did not specify optionValues",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
						Option:      &optionSpecThree,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template: &jobTemplateSpecBasic.Spec,
				},
			},
			wantErrors: "spec.optionValues[option1]: Required value: option is required",
		},
		{
			name: "required option with no value",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
						Option:      &optionSpecThree,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template: &jobTemplateSpecBasic.Spec,
					OptionValues: optionValues(map[string]interface{}{
						"option2": "newvalue2",
					}),
				},
			},
			wantErrors: "spec.optionValues[option1]: Required value: option is required",
		},
		{
			name: "option with wrong type",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
						Option:      &optionSpecThree,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template: &jobTemplateSpecBasic.Spec,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "1",
						"option2": 2,
					}),
				},
			},
			wantErrors: "spec.optionValues[option2]: Invalid value: 2: expected string, got float64",
		},
		{
			name: "mutate ConfigName",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					ConfigName: objectMetaJobConfig.Name,
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
				},
			},
		},
		{
			name: "mutate ConfigName with custom startPolicy",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					ConfigName:  objectMetaJobConfig.Name,
					StartPolicy: &startPolicyOverride,
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyOverride,
				},
			},
		},
		{
			name: "mutate ConfigName with substitutions",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
						Option:      &optionSpecThree,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					ConfigName:  objectMetaJobConfig.Name,
					StartPolicy: &startPolicyBasic,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "new_value",
					}),
					Substitutions: map[string]string{
						"option.option1": "new_value2",
					},
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithOptionsHash,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "new_value",
					}),
					Substitutions: map[string]string{
						"option.option1": "new_value2",
						"option.option2": "value2",
						"option.option3": "value3",
					},
				},
			},
		},
		{
			name: "mutate ConfigName with optionValues",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
						Option:      &optionSpecThree,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					ConfigName:  objectMetaJobConfig.Name,
					StartPolicy: &startPolicyBasic,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "newvalue1",
						"option2": "newvalue2",
					}),
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithOptionsHash,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "newvalue1",
						"option2": "newvalue2",
					}),
					Substitutions: map[string]string{
						"option.option1": "newvalue1",
						"option.option2": "newvalue2",
						"option.option3": "value3",
					},
				},
			},
		},
		{
			name: "mutate ConfigName with optionValues and substitutions",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
						Option:      &optionSpecThree,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					ConfigName:  objectMetaJobConfig.Name,
					StartPolicy: &startPolicyBasic,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "newvalue1",
						"option2": "newvalue2",
					}),
					Substitutions: map[string]string{
						"option.option2": "substitutionvalue2",
					},
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithOptionsHash,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
					OptionValues: optionValues(map[string]interface{}{
						"option1": "newvalue1",
						"option2": "newvalue2",
					}),
					Substitutions: map[string]string{
						"option.option1": "newvalue1",
						"option.option2": "substitutionvalue2",
						"option.option3": "value3",
					},
				},
			},
		},
		{
			name: "mutate ConfigName with warnings",
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					ConfigName: objectMetaJobConfig.Name,
					Template: &v1alpha1.JobTemplateSpec{
						Task: v1alpha1.TaskSpec{
							Template: v1alpha1.TaskTemplate{
								Pod: &podTemplateSpecBasic,
							},
							ForbidForceDeletion: true,
						},
					},
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
				},
			},
			wantWarnings: []string{"JobTemplate was overwritten with JobConfig's template"},
		},
		{
			name: "mutate ConfigName cannot get JobConfig",
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJob,
				Spec: v1alpha1.JobSpec{
					ConfigName: objectMetaJobConfig.Name,
				},
			},
			wantErrors: "spec.configName: Not found: \"jobconfig-sample\"",
		},
		{
			name: "add jobconfig context variables",
			setup: func() {
				variablecontext.ContextProvider = &mockProvider{
					ProvideForJobConfig: func(rjc *v1alpha1.JobConfig) map[string]string {
						return map[string]string{
							"jobconfig.provider_value_1": "default_provider_value_1",
							"jobconfig.provider_value_2": "default_provider_value_2",
						}
					},
				}
			},
			rjcs: []*v1alpha1.JobConfig{
				{
					ObjectMeta: objectMetaJobConfig,
					Spec: v1alpha1.JobConfigSpec{
						Template:    jobTemplateSpecBasic,
						Concurrency: concurrencySpecBasic,
					},
				},
			},
			rj: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
					Substitutions: map[string]string{
						"jobconfig.custom_substitution": "my_substitution",
						"jobconfig.provider_value_1":    "custom_provider_value_1",
					},
				},
			},
			want: &v1alpha1.Job{
				ObjectMeta: objectMetaJobWithAllReferences,
				Spec: v1alpha1.JobSpec{
					Template:    &jobTemplateSpecBasic.Spec,
					StartPolicy: &startPolicyBasic,
					Substitutions: map[string]string{
						"jobconfig.custom_substitution": "my_substitution",
						"jobconfig.provider_value_1":    "custom_provider_value_1",
						"jobconfig.provider_value_2":    "default_provider_value_2",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutator := setup(t, tt.cfgs, tt.rjcs)
			if tt.setup != nil {
				tt.setup()
			}

			newRj := tt.rj.DeepCopy()
			resp := mutator.MutateCreateJob(newRj)
			if err := checkResult(resp, tt.wantErrors, tt.wantWarnings); err != "" {
				t.Errorf("MutateCreateJob() %v", err)
			}
			if tt.wantErrors != "" {
				return
			}

			opts := []cmp.Option{cmpopts.EquateEmpty()}
			if tt.want == nil {
				if !cmp.Equal(newRj, tt.rj, opts...) {
					t.Errorf("MutateCreateJob() expected no change\ndiff = %v", cmp.Diff(tt.rj, newRj, opts...))
				}
			} else if !cmp.Equal(newRj, tt.want, opts...) {
				t.Errorf("MutateCreateJob() not equal\ndiff = %v", cmp.Diff(tt.want, newRj, opts...))
			}
		})
	}
}

type noopProvider struct{}

func (n *noopProvider) GetAllPrefixes() []string {
	return nil
}

func (n *noopProvider) MakeVariablesFromJobConfig(rjc *v1alpha1.JobConfig) map[string]string {
	return nil
}

func (n *noopProvider) MakeVariablesFromJob(rj *v1alpha1.Job) map[string]string {
	return nil
}

func (n *noopProvider) MakeVariablesFromTask(task variablecontext.TaskSpec) map[string]string {
	return nil
}

type mockProvider struct {
	ProvideForJobConfig func(rjc *v1alpha1.JobConfig) map[string]string
}

func (m *mockProvider) GetAllPrefixes() []string {
	return nil
}

func (m *mockProvider) MakeVariablesFromJobConfig(rjc *v1alpha1.JobConfig) map[string]string {
	return m.ProvideForJobConfig(rjc)
}

func (m *mockProvider) MakeVariablesFromJob(rj *v1alpha1.Job) map[string]string {
	return nil
}

func (m *mockProvider) MakeVariablesFromTask(task variablecontext.TaskSpec) map[string]string {
	return nil
}

func setupContext(t *testing.T, cfgs map[configv1alpha1.ConfigName]runtime.Object, rjcs []*v1alpha1.JobConfig) controllercontext.Context {
	ctx := context.Background()
	ctrlContext := mock.NewContext()
	ctrlContext.MockConfigs().SetConfigs(cfgs)
	hasSynced := ctrlContext.Informers().Furiko().Execution().V1alpha1().JobConfigs().Informer().HasSynced
	if err := ctrlContext.Start(ctx); err != nil {
		t.Fatal(err)
	}

	for _, rjc := range rjcs {
		_, err := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1().JobConfigs(rjc.Namespace).
			Create(ctx, rjc, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("cannot create JobConfig: %v", err)
		}
	}

	if !cache.WaitForCacheSync(ctx.Done(), hasSynced) {
		t.Fatalf("cannot sync caches")
	}

	// Replace provider
	variablecontext.ContextProvider = &noopProvider{}

	return ctrlContext
}

func setup(t *testing.T, cfgs map[configv1alpha1.ConfigName]runtime.Object, rjcs []*v1alpha1.JobConfig) *mutation.Mutator {
	ctrlContext := setupContext(t, cfgs, rjcs)
	return mutation.NewMutator(ctrlContext)
}

func checkResult(result *webhook.Result, wantErrors string, wantWarnings []string) string {
	err := result.Errors.ToAggregate()

	// Mismatched error
	if (err == nil) != (wantErrors == "") {
		return fmt.Sprintf("error = %v, wantErrors = %v", err, wantErrors)
	}

	// Wrong error
	if err != nil && !strings.HasPrefix(err.Error(), wantErrors) {
		return fmt.Sprintf("error = %v, wantErrors = %v", err, wantErrors)
	}

	// Mismatched warnings
	if !cmp.Equal(result.Warnings, wantWarnings) {
		return fmt.Sprintf("warnings = %v, wantWarnings = %v", result.Warnings, wantWarnings)
	}

	return ""
}

func optionValuesYAML(values map[string]interface{}) string {
	bytes, err := yaml.Marshal(values)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func optionValues(values map[string]interface{}) string {
	bytes, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func mkScheduleSpec(expr string, lastUpdated *metav1.Time) *v1alpha1.ScheduleSpec {
	scheduleSpec := &v1alpha1.ScheduleSpec{
		Cron: &v1alpha1.CronSchedule{
			Expression: expr,
			Timezone:   "Asia/Singapore",
		},
		LastUpdated: lastUpdated,
	}
	return scheduleSpec
}
