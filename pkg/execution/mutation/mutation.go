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
package mutation

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	"github.com/furiko-io/furiko/apis/execution"
	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
	executionlister "github.com/furiko-io/furiko/pkg/generated/listers/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/utils/cmp"
	"github.com/furiko-io/furiko/pkg/utils/execution/jobconfig"
	"github.com/furiko-io/furiko/pkg/utils/jsonyaml"
	"github.com/furiko-io/furiko/pkg/utils/k8sutils"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/webhook"
)

var (
	Clock clock.Clock = &clock.RealClock{}
)

// Mutator encapsulates all mutation methods.
type Mutator struct {
	ctrlContext controllercontext.Context
}

func NewMutator(ctrlContext controllercontext.Context) *Mutator {
	return &Mutator{ctrlContext: ctrlContext}
}

// MutateJobConfig mutates a v1alpha1.JobConfig in-place.
func (m *Mutator) MutateJobConfig(rjc *v1alpha1.JobConfig) *webhook.Result {
	result := webhook.NewResult()

	// Specify default values for JobTemplate.
	result.Merge(m.MutateJobTemplate(&rjc.Spec.Template,
		field.NewPath("spec").Child("spec").Child("template")))

	// Specify default values for OptionSpec.
	if spec := rjc.Spec.Option; spec != nil {
		rjc.Spec.Option = options.DefaultOptionSpec(spec)
	}

	return result
}

// MutateCreateJobConfig mutates a v1alpha1.JobConfig in-place for creation.
func (m *Mutator) MutateCreateJobConfig(rjc *v1alpha1.JobConfig) *webhook.Result {
	result := webhook.NewResult()

	// Bump NotBefore when JobConfig is first created.
	// NOTE(irvinlim): Do not create scheduleSpec if nil.
	now := metav1.NewTime(Clock.Now())
	if rjc.Spec.Schedule != nil {
		if rjc.Spec.Schedule.Constraints == nil {
			rjc.Spec.Schedule.Constraints = &v1alpha1.ScheduleContraints{}
		}
		if !ktime.IsTimeSetAndLaterThan(rjc.Spec.Schedule.Constraints.NotBefore, now.Time) {
			rjc.Spec.Schedule.Constraints.NotBefore = &now
		}
	}

	return result
}

// MutateUpdateJobConfig mutates a v1alpha1.JobConfig in-place for update.
func (m *Mutator) MutateUpdateJobConfig(oldRjc, rjc *v1alpha1.JobConfig) *webhook.Result {
	result := webhook.NewResult()
	now := metav1.NewTime(Clock.Now())

	// Bump NotBefore if the cron schedules were updated.
	if rjc.Spec.Schedule != nil {
		var oldCron *v1alpha1.CronSchedule
		if spec := oldRjc.Spec.Schedule; spec != nil {
			oldCron = spec.Cron
		}
		isEqual, err := cmp.IsJSONEqual(oldCron, rjc.Spec.Schedule.Cron)
		if err != nil {
			fldPath := field.NewPath("spec").Child("schedule").Child("cron")
			result.Errors = append(result.Errors, field.InternalError(fldPath, err))
		} else if !isEqual {
			if rjc.Spec.Schedule.Constraints == nil {
				rjc.Spec.Schedule.Constraints = &v1alpha1.ScheduleContraints{}
			}
			if !ktime.IsTimeSetAndLaterThan(rjc.Spec.Schedule.Constraints.NotBefore, now.Time) {
				rjc.Spec.Schedule.Constraints.NotBefore = &now
			}
		}
	}

	return result
}

// MutateJob mutates a v1alpha1.Job in-place.
func (m *Mutator) MutateJob(rj *v1alpha1.Job) *webhook.Result {
	result := webhook.NewResult()

	cfg, err := m.ctrlContext.Configs().Jobs()
	if err != nil {
		result.Errors = append(result.Errors, field.InternalError(field.NewPath(""), err))
		return result
	}

	// Add default values.
	if rj.Spec.Type == "" {
		rj.Spec.Type = v1alpha1.JobTypeAdhoc
	}
	if rj.Spec.TTLSecondsAfterFinished == nil {
		rj.Spec.TTLSecondsAfterFinished = cfg.DefaultTTLSecondsAfterFinished
	}
	if rj.Spec.Template.MaxAttempts == nil {
		rj.Spec.Template.MaxAttempts = pointer.Int32(1)
	}

	return result
}

// MutateCreateJob mutates a v1alpha1.Job in-place for creation.
func (m *Mutator) MutateCreateJob(rj *v1alpha1.Job) *webhook.Result {
	result := webhook.NewResult()

	// Add default finalizers if not set.
	// NOTE(irvinlim): We should not add on update, because it affects deletion.
	if !k8sutils.ContainsFinalizer(rj.Finalizers, execution.DeleteDependentsFinalizer) {
		rj.Finalizers = k8sutils.MergeFinalizers(rj.Finalizers, []string{execution.DeleteDependentsFinalizer})
	}

	// Evaluate configName and populate fields from the JobConfig.
	result.Merge(m.evaluateConfigName(rj, rj.Spec.ConfigName, field.NewPath("spec").Child("configName")))

	// Look up the JobConfig or fail validation if we retrieved an invalid JobConfig owner.
	rjc, errs := jobconfig.ValidateLookupJobOwner(rj, m.getJobConfigLister(rj.Namespace))
	if errs != nil {
		result.Errors = append(result.Errors, errs...)
		return result
	}

	// Evaluate all job options and validate them before admitting the job.
	result.Merge(m.evaluateOptionValues(rj, rjc, field.NewPath("spec").Child("optionValues")))

	// Add JobConfig context variables, but let the spec take precedence.
	if rjc != nil {
		rj.Spec.Substitutions = options.MergeSubstitutions(
			variablecontext.ContextProvider.MakeVariablesFromJobConfig(rjc),
			rj.Spec.Substitutions,
		)
	}

	return result
}

// evaluateOptionValues mutates the v1alpha1.Job in-place after evaluating job
// options, and returns any validation errors encountered.
func (m *Mutator) evaluateConfigName(rj *v1alpha1.Job, rjcName string, fldPath *field.Path) *webhook.Result {
	result := webhook.NewResult()

	// No need to evaluate anything.
	if rjcName == "" {
		return result
	}

	// Get JobConfig from informer.
	rjc, err := m.getJobConfigLister(rj.Namespace).Get(rjcName)
	if err != nil {
		if kerrors.IsNotFound(err) {
			result.Errors = append(result.Errors, field.NotFound(fldPath, rjcName))
		} else {
			result.Errors = append(result.Errors, field.InternalError(fldPath, err))
		}

		return result
	}

	// Create a base Job from the JobConfig.
	// Any variables specified in the spec will take higher precedence.
	baseJob, err := jobconfig.NewJobFromJobConfig(rjc, rj.Spec.Type, rj.CreationTimestamp.Time)
	if err != nil {
		result.Errors = append(result.Errors, field.InternalError(fldPath,
			errors.Wrapf(err, "cannot create job from jobconfig")))
		return result
	}

	// Merge metadata fields down, with the incoming Job's metadata taking precedence.
	rj.Labels = labels.Merge(baseJob.Labels, rj.Labels)
	rj.Annotations = labels.Merge(baseJob.Annotations, rj.Annotations)
	rj.Finalizers = k8sutils.MergeFinalizers(baseJob.Finalizers, rj.Finalizers)

	// Ensure that JobConfig references are intact.
	rj.OwnerReferences = baseJob.OwnerReferences
	rj.Labels[jobconfig.LabelKeyJobConfigUID] = baseJob.Labels[jobconfig.LabelKeyJobConfigUID]

	// Return Warnings if the JobTemplate was not empty when overriding.
	if rj.Spec.Template != nil && !reflect.DeepEqual(rj.Spec.Template, v1alpha1.JobTemplateSpec{}) {
		result.Warnings = append(result.Warnings, "JobTemplate was overwritten with JobConfig's template")
	}

	// The JobTemplate will be completely overwritten by the base Job.
	rj.Spec.Template = baseJob.Spec.Template

	// If StartPolicy is not set, we should default to the JobConfig's ConcurrencyPolicy.
	if startPolicy := rj.Spec.StartPolicy; startPolicy == nil {
		rj.Spec.StartPolicy = &v1alpha1.StartPolicySpec{}
	}
	if cp := rj.Spec.StartPolicy.ConcurrencyPolicy; cp == "" {
		rj.Spec.StartPolicy.ConcurrencyPolicy = rjc.Spec.Concurrency.Policy
	}

	// Finally, unset the ConfigName from the spec because it should not be saved in the spec.
	rj.Spec.ConfigName = ""

	return result
}

// evaluateOptionValues mutates the v1alpha1.Job in-place after evaluating job
// options, and returns any validation errors encountered.
func (m *Mutator) evaluateOptionValues(rj *v1alpha1.Job, rjc *v1alpha1.JobConfig, fldPath *field.Path) *webhook.Result {
	result := webhook.NewResult()
	optionValues := make(map[string]interface{})

	// There is no parent JobConfig defined. We cannot evaluate any option values or defaults whatsoever.
	if rjc == nil {
		// If optionValues is defined, this is a no-op. We should return warnings.
		if len(rj.Spec.OptionValues) > 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("optionValues will be ignored without a JobConfig: %v", rj.Spec.OptionValues))
		}
		return result
	}

	if len(rj.Spec.OptionValues) > 0 {
		// Unmarshal into map[string]interface{} here. We cannot use this type in the
		// CRD otherwise deepcopy-gen will complain.
		if err := jsonyaml.UnmarshalString(rj.Spec.OptionValues, &optionValues); err != nil {
			err := errors.Wrapf(err, "cannot unmarshal as json or yaml")
			result.Errors = append(result.Errors, field.Invalid(fldPath, rj.Spec.OptionValues, err.Error()))
			return result
		}

		// Re-marshal into JSON to standardize the output value (mostly useful to avoid
		// corrupting `kubectl describe`).
		jsonValues, err := json.Marshal(optionValues)
		if err != nil {
			result.Errors = append(result.Errors, field.InternalError(fldPath, err))
			return result
		}
		rj.Spec.OptionValues = string(jsonValues)

		// Store a hash of the OptionSpec in the annotations as well. This is useful
		// to determine if the OptionSpec had been modified between two runs of the
		// same JobConfig, and if so, re-using the OptionValues for a subsequent run may
		// be invalid.
		if hash, err := options.HashOptionSpec(rjc.Spec.Option); err != nil {
			warning := fmt.Sprintf("failed to store option spec hash in annotations: %v", err)
			result.Warnings = append(result.Warnings, warning)
		} else {
			k8sutils.SetAnnotation(rj, jobconfig.AnnotationKeyOptionSpecHash, hash)
		}
	}

	// Evaluate the option values. If any required option is empty, an error will be raised.
	// TODO(irvinlim): Need to fail admission if OptionValues contains a non-existent option name.
	evaluated, errs := options.EvaluateOptions(optionValues, rjc.Spec.Option, fldPath)
	if len(errs) > 0 {
		result.Errors = append(result.Errors, errs...)
		return result
	}

	// Merge the substitutions. The order of priority from lowest to highest is:
	// JobConfig's default options -> Evaluated options -> Explicitly specified substitutions
	rj.Spec.Substitutions = options.MergeSubstitutions(evaluated, rj.Spec.Substitutions)

	return result
}

func (m *Mutator) MutateJobTemplate(spec *v1alpha1.JobTemplate, fldPath *field.Path) *webhook.Result {
	result := webhook.NewResult()

	// Specify default RestartPolicy.
	if spec.Spec.Task.Template.Spec.RestartPolicy == "" {
		spec.Spec.Task.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	}

	return result
}

func (m *Mutator) getJobConfigLister(namespace string) executionlister.JobConfigNamespaceLister {
	return m.ctrlContext.Informers().Furiko().Execution().V1alpha1().JobConfigs().Lister().
		JobConfigs(namespace)
}
