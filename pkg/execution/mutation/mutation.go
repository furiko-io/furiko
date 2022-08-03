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
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	"github.com/furiko-io/furiko/apis/execution"
	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
	executionlister "github.com/furiko-io/furiko/pkg/generated/listers/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/webhook"
	"github.com/furiko-io/furiko/pkg/utils/jsonyaml"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/meta"
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

	// Specify default values for OptionSpec.
	if spec := rjc.Spec.Option; spec != nil {
		rjc.Spec.Option = options.MutateDefaultingOptionSpec(spec)
	}

	// Mutate JobTemplate, but don't add taskTemplate fields, because they should be
	// added only when the Job is created.
	result.Merge(m.MutateJobTemplateSpec(&rjc.Spec.Template.Spec, false,
		field.NewPath("spec", "template", "spec")))

	return result
}

// MutateCreateJobConfig mutates a v1alpha1.JobConfig in-place for creation.
func (m *Mutator) MutateCreateJobConfig(rjc *v1alpha1.JobConfig) *webhook.Result {
	result := webhook.NewResult()

	// Bump LastUpdated when JobConfig is first created.
	// NOTE(irvinlim): Do not create scheduleSpec if nil.
	now := metav1.NewTime(Clock.Now())
	if rjc.Spec.Schedule != nil {
		if !ktime.IsTimeSetAndLaterThan(rjc.Spec.Schedule.LastUpdated, now.Time) {
			rjc.Spec.Schedule.LastUpdated = &now
		}
	}

	return result
}

// MutateUpdateJobConfig mutates a v1alpha1.JobConfig in-place for update.
func (m *Mutator) MutateUpdateJobConfig(oldRjc, rjc *v1alpha1.JobConfig) *webhook.Result {
	result := webhook.NewResult()
	now := metav1.NewTime(Clock.Now())

	// Bump LastUpdated if the schedule was updated.
	if rjc.Spec.Schedule != nil {
		newSchedule := rjc.Spec.Schedule.DeepCopy()
		if oldRjc.Spec.Schedule != nil {
			newSchedule.LastUpdated = oldRjc.Spec.Schedule.LastUpdated
		}
		if !apiequality.Semantic.DeepEqual(oldRjc.Spec.Schedule, newSchedule) {
			if !ktime.IsTimeSetAndLaterThan(rjc.Spec.Schedule.LastUpdated, now.Time) {
				rjc.Spec.Schedule.LastUpdated = &now
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

	// Specify default values for JobTemplate.
	if rj.Spec.Template == nil {
		rj.Spec.Template = &v1alpha1.JobTemplate{}
	}
	result.Merge(m.MutateJobTemplateSpec(rj.Spec.Template, true, field.NewPath("spec", "template")))

	return result
}

// MutateCreateJob mutates a v1alpha1.Job in-place for creation.
func (m *Mutator) MutateCreateJob(rj *v1alpha1.Job) *webhook.Result {
	result := webhook.NewResult()

	// Add default finalizers if not set.
	// NOTE(irvinlim): We should not add on update, because it affects deletion.
	if !meta.ContainsFinalizer(rj.Finalizers, execution.DeleteDependentsFinalizer) {
		rj.Finalizers = meta.MergeFinalizers(rj.Finalizers, []string{execution.DeleteDependentsFinalizer})
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
	rj.Finalizers = meta.MergeFinalizers(baseJob.Finalizers, rj.Finalizers)

	// Ensure that JobConfig references are intact.
	rj.OwnerReferences = baseJob.OwnerReferences
	rj.Labels[jobconfig.LabelKeyJobConfigUID] = baseJob.Labels[jobconfig.LabelKeyJobConfigUID]

	// Return Warnings if the JobTemplateSpec was not empty when overriding.
	if rj.Spec.Template != nil && !reflect.DeepEqual(rj.Spec.Template, v1alpha1.JobTemplate{}) {
		result.Warnings = append(result.Warnings, "JobTemplateSpec was overwritten with JobConfig's template")
	}

	// The JobTemplateSpec will be completely overwritten by the base Job.
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
			meta.SetAnnotation(rj, jobconfig.AnnotationKeyOptionSpecHash, hash)
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

// MutateJobTemplateSpec mutates a JobTemplate in-place.
func (m *Mutator) MutateJobTemplateSpec(
	spec *v1alpha1.JobTemplate,
	mutateTaskTemplate bool,
	fldPath *field.Path,
) *webhook.Result {
	result := webhook.NewResult()

	if spec.MaxAttempts == nil {
		spec.MaxAttempts = pointer.Int64(1)
	}
	if spec.Parallelism != nil {
		result.Merge(m.MutateParallelismSpec(spec.Parallelism, fldPath.Child("parallelism")))
	}
	result.Merge(m.ValidateTaskTemplate(&spec.TaskTemplate, fldPath.Child("taskTemplate")))
	if mutateTaskTemplate {
		result.Merge(m.MutateTaskTemplate(&spec.TaskTemplate, fldPath.Child("taskTemplate")))
	}

	return result
}

// ValidateTaskTemplate performs validation for a TaskTemplate. Mainly used for returning warnings.
func (m *Mutator) ValidateTaskTemplate(spec *v1alpha1.TaskTemplate, fldPath *field.Path) *webhook.Result {
	result := webhook.NewResult()
	if spec.ArgoWorkflow != nil {
		result.Merge(m.ValidateArgoWorkflowTemplateSpec(spec.ArgoWorkflow, fldPath.Child("argoWorkflow")))
	}
	return result
}

// ValidateArgoWorkflowTemplateSpec mutates ArgoWorkflowTemplateSpec in-place.
func (m *Mutator) ValidateArgoWorkflowTemplateSpec(spec *v1alpha1.ArgoWorkflowTemplateSpec, fldPath *field.Path) *webhook.Result {
	result := webhook.NewResult()

	// Did not find Argo Workflow CRD, but user is trying to mutate a Job/JobConfig
	// with an argoWorkflow task template.
	if _, err := m.ctrlContext.CustomResourceDefinitions().ArgoWorkflows(); err != nil {
		result.Warnings = append(result.Warnings, "Argo Workflows could not be detected in the cluster, the job may fail to run")
	}

	return result
}

// MutateTaskTemplate mutates a TaskTemplate in-place.
func (m *Mutator) MutateTaskTemplate(spec *v1alpha1.TaskTemplate, fldPath *field.Path) *webhook.Result {
	result := webhook.NewResult()
	if spec.Pod != nil {
		result.Merge(m.MutatePodTemplateSpec(spec.Pod, fldPath.Child("pod")))
	}
	return result
}

// MutateParallelismSpec mutates ParallelismSpec in-place.
func (m *Mutator) MutateParallelismSpec(spec *v1alpha1.ParallelismSpec, fldPath *field.Path) *webhook.Result {
	result := webhook.NewResult()

	if spec.CompletionStrategy == "" {
		spec.CompletionStrategy = v1alpha1.AllSuccessful
	}

	return result
}

// MutatePodTemplateSpec mutates PodTemplateSpec in-place.
func (m *Mutator) MutatePodTemplateSpec(spec *v1alpha1.PodTemplateSpec, fldPath *field.Path) *webhook.Result {
	result := webhook.NewResult()

	if spec.Spec.RestartPolicy == "" {
		spec.Spec.RestartPolicy = corev1.RestartPolicyNever
	}

	return result
}

func (m *Mutator) getJobConfigLister(namespace string) executionlister.JobConfigNamespaceLister {
	return m.ctrlContext.Informers().Furiko().Execution().V1alpha1().JobConfigs().Lister().
		JobConfigs(namespace)
}
