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
package validation

import (
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	apimachineryvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	apivalidation "k8s.io/kubernetes/pkg/apis/core/validation"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
	"github.com/furiko-io/furiko/pkg/core/tzutils"
	"github.com/furiko-io/furiko/pkg/core/validation"
	"github.com/furiko-io/furiko/pkg/execution/util/cronparser"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
	executionlister "github.com/furiko-io/furiko/pkg/generated/listers/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

const (
	// The JobConfig creates Jobs with a suffix like `.1646586360` and each Job has
	// a suffix with the retry index like `.20`. In total, the suffix length may be
	// 14 bytes, so we have to limit the JobConfig's name to be 63-14 = 49
	// characters.
	maxJobConfigNameLen = apimachineryvalidation.DNS1035LabelMaxLength - 14

	// The Job creates tasks with a suffix like `.20`, so the name has to be 60
	// characters.
	maxJobNameLen = apimachineryvalidation.DNS1035LabelMaxLength - 3

	errMessageTooManyTemplates = "cannot specify more than 1 template type"
)

var (
	Clock clock.Clock = &clock.RealClock{}
)

// Validator encapsulates all validator methods.
type Validator struct {
	ctrlContext controllercontext.Context
}

func NewValidator(ctrlContext controllercontext.Context) *Validator {
	return &Validator{ctrlContext: ctrlContext}
}

// ValidateJobConfig validates a *v1alpha1.JobConfig.
func (v *Validator) ValidateJobConfig(rjc *v1alpha1.JobConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validation.ValidateMaxLength(rjc.Name, maxJobConfigNameLen, field.NewPath("metadata").Child("name"))...)
	allErrs = append(allErrs, v.ValidateJobConfigSpec(&rjc.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateJobConfigCreate validates creation of a *v1alpha1.JobConfig.
func (v *Validator) ValidateJobConfigCreate(rjc *v1alpha1.JobConfig) field.ErrorList {
	// No special handling.
	return nil
}

// ValidateJobConfigUpdate validates update of a *v1alpha1.JobConfig.
func (v *Validator) ValidateJobConfigUpdate(oldRjc, rjc *v1alpha1.JobConfig) field.ErrorList {
	// No special handling.
	return nil
}

// ValidateJob validates a *v1alpha1.Job.
func (v *Validator) ValidateJob(rj *v1alpha1.Job) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, v.ValidateJobMetadata(&rj.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, v.ValidateJobSpec(&rj.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateJobMetadata validates the metadata of a *v1alpha1.Job.
func (v *Validator) ValidateJobMetadata(metadata *metav1.ObjectMeta, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validation.ValidateMaxLength(metadata.Name, maxJobNameLen, fldPath.Child("name"))...)
	return allErrs
}

// ValidateJobCreate validates creation of a *v1alpha1.Job with the parent JobConfig.
func (v *Validator) ValidateJobCreate(rj *v1alpha1.Job) field.ErrorList {
	// Look up JobConfig for the job.
	rjc, errs := jobconfig.ValidateLookupJobOwner(rj, v.getJobConfigLister(rj.Namespace))
	if errs != nil {
		return errs
	}

	cfg, err := v.ctrlContext.Configs().JobConfigs()
	if err != nil {
		return field.ErrorList{
			field.InternalError(field.NewPath(""), errors.Wrapf(err, "cannot load config")),
		}
	}

	// Validation specific to case when it belongs to JobConfig.
	if rjc != nil {
		return v.validateJobCreateWithJobConfig(rj, rjc, cfg)
	}

	return nil
}

func (v *Validator) validateJobCreateWithJobConfig(
	rj *v1alpha1.Job,
	rjc *v1alpha1.JobConfig,
	cfg *configv1alpha1.JobConfigExecutionConfig,
) field.ErrorList {
	allErrs := field.ErrorList{}

	// Reject Job if ConcurrencyPolicyForbid and JobConfig has active Jobs.
	if spec := rj.Spec.StartPolicy; spec != nil && rjc != nil &&
		spec.ConcurrencyPolicy == v1alpha1.ConcurrencyPolicyForbid &&
		rjc.Status.Active > 0 {
		allErrs = append(allErrs, field.Forbidden(
			field.NewPath("spec.startPolicy.concurrencyPolicy"),
			fmt.Sprintf(
				"cannot create new Job for JobConfig %v, concurrencyPolicy is Forbid but there are %v active jobs",
				rjc.Name, rjc.Status.Active,
			),
		))
	}

	// Cannot enqueue beyond max queue length.
	if max := cfg.MaxEnqueuedJobs; max != nil && rjc != nil && rjc.Status.Queued >= *max {
		allErrs = append(allErrs, field.Forbidden(
			field.NewPath("spec.startPolicy"),
			fmt.Sprintf("cannot create new Job for JobConfig %v, which would exceed maximum queue length of %v", rjc.Name, *max),
		))
	}

	return allErrs
}

// ValidateJobUpdate validates update of a *v1alpha1.Job.
func (v *Validator) ValidateJobUpdate(oldRj, rj *v1alpha1.Job) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, v.ValidateJobMetadataUpdate(&oldRj.ObjectMeta, &rj.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, v.ValidateJobSpecUpdate(&oldRj.Spec, &rj.Spec, field.NewPath("spec"))...)

	// Once Job is started, not allowed to update startPolicy.
	if !rj.Status.StartTime.IsZero() {
		allErrs = append(allErrs, validation.ValidateImmutableField(rj.Spec.StartPolicy, oldRj.Spec.StartPolicy,
			field.NewPath("spec.startPolicy"), "cannot update startPolicy once Job is started")...)
	}

	return allErrs
}

// ValidateJobMetadataUpdate validates update of the metadata of a *v1alpha1.Job.
func (v *Validator) ValidateJobMetadataUpdate(oldMetadata, metadata *metav1.ObjectMeta, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Cannot update JobConfig UID label.
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(
		metadata.Labels[jobconfig.LabelKeyJobConfigUID],
		oldMetadata.Labels[jobconfig.LabelKeyJobConfigUID],
		fldPath.Child("labels").Key(jobconfig.LabelKeyJobConfigUID),
	)...)

	return allErrs
}

// ValidateJobConfigSpec validates a *v1alpha1.JobConfigSpec.
func (v *Validator) ValidateJobConfigSpec(spec *v1alpha1.JobConfigSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, v.ValidateJobTemplate(&spec.Template, fldPath.Child("template"))...)
	allErrs = append(allErrs, v.ValidateConcurrencySpec(spec.Concurrency, fldPath.Child("concurrency"))...)
	allErrs = append(allErrs, v.ValidateScheduleSpec(spec.Schedule, fldPath.Child("schedule"))...)
	allErrs = append(allErrs, v.ValidateOptionSpec(spec.Option, fldPath.Child("option"))...)
	return allErrs
}

// ValidateJobTemplate validates a *v1alpha1.JobTemplate.
func (v *Validator) ValidateJobTemplate(spec *v1alpha1.JobTemplate, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, v.ValidateJobTemplateSpec(&spec.Spec, fldPath.Child("spec"))...)
	return allErrs
}

// ValidateConcurrencySpec validates a v1alpha1.ConcurrencySpec.
func (v *Validator) ValidateConcurrencySpec(spec v1alpha1.ConcurrencySpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, v.ValidateConcurrencyPolicy(spec.Policy, fldPath.Child("policy"))...)
	return allErrs
}

// ValidateScheduleSpec validates a *v1alpha1.ScheduleSpec.
func (v *Validator) ValidateScheduleSpec(spec *v1alpha1.ScheduleSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec == nil {
		return allErrs
	}

	var numScheduleTypes int
	if spec.Cron != nil {
		if numScheduleTypes > 1 {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("cron"), "may not specify more than 1 schedule type"))
		} else {
			numScheduleTypes++
			allErrs = append(allErrs, v.ValidateCronSchedule(spec.Cron, fldPath.Child("cron"))...)
		}
	}

	if numScheduleTypes < 1 {
		allErrs = append(allErrs, field.Required(fldPath, "at least one schedule type must be specified"))
	}

	return allErrs
}

// ValidateCronSchedule validates a *v1alpha1.CronSchedule.
func (v *Validator) ValidateCronSchedule(spec *v1alpha1.CronSchedule, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(spec.Expression) > 0 {
		allErrs = append(allErrs, v.ValidateCronScheduleExpression(spec.Expression, fldPath.Child("expression"))...)
	}
	if len(spec.Timezone) > 0 {
		allErrs = append(allErrs, v.ValidateTimezone(spec.Timezone, fldPath.Child("timezone"))...)
	}
	return allErrs
}

// ValidateConcurrencyPolicy validates a v1alpha1.ConcurrencyPolicy.
func (v *Validator) ValidateConcurrencyPolicy(concurrencyPolicy v1alpha1.ConcurrencyPolicy, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	switch concurrencyPolicy {
	case v1alpha1.ConcurrencyPolicyAllow,
		v1alpha1.ConcurrencyPolicyForbid,
		v1alpha1.ConcurrencyPolicyEnqueue:
		break
	case "":
		allErrs = append(allErrs, field.Required(fldPath, ""))
	default:
		validValues := []string{
			string(v1alpha1.ConcurrencyPolicyAllow),
			string(v1alpha1.ConcurrencyPolicyForbid),
			string(v1alpha1.ConcurrencyPolicyEnqueue),
		}
		allErrs = append(allErrs, field.NotSupported(fldPath, concurrencyPolicy, validValues))
	}
	return allErrs
}

// ValidateCronScheduleExpression validates a CronSchedule expression.
func (v *Validator) ValidateCronScheduleExpression(cronSchedule string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Load the Cron config to determine how to parse the cron expression.
	cfg, err := v.ctrlContext.Configs().Cron()
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath, errors.Wrapf(err, "cannot load cron config")))
		return allErrs
	}
	parser := cronparser.NewParser(cfg)

	// Ok to use an empty hash ID for validation.
	if _, err := parser.Parse(cronSchedule, ""); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, cronSchedule, "cannot parse cron schedule"))
	}

	return allErrs
}

// ValidateTimezone validates a Timezone.
func (v *Validator) ValidateTimezone(timezone string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := tzutils.ParseTimezone(timezone); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, timezone, "cannot parse timezone"))
	}
	return allErrs
}

// ValidateOptionSpec validates a *v1alpha1.OptionSpec.
func (v *Validator) ValidateOptionSpec(spec *v1alpha1.OptionSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec != nil {
		allErrs = append(allErrs, options.ValidateOptionSpec(spec, fldPath)...)
	}
	return allErrs
}

// ValidateJobSpec validates a *v1alpha1.JobSpec.
func (v *Validator) ValidateJobSpec(spec *v1alpha1.JobSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, v.ValidateJobType(spec.Type, fldPath.Child("type"))...)
	allErrs = append(allErrs, v.ValidateStartPolicySpec(spec.StartPolicy, fldPath.Child("startPolicy"))...)
	if spec.Template == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("template"), ""))
	} else {
		allErrs = append(allErrs, v.ValidateJobTemplateSpec(spec.Template, fldPath.Child("template"))...)
	}
	if spec.TTLSecondsAfterFinished != nil {
		allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(*spec.TTLSecondsAfterFinished, fldPath.Child("ttlSecondsAfterFinished"))...)
	}
	return allErrs
}

// ValidateJobSpecUpdate validates update of a *v1alpha1.JobSpec.
func (v *Validator) ValidateJobSpecUpdate(oldSpec, spec *v1alpha1.JobSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(spec.ConfigName, oldSpec.ConfigName, fldPath.Child("configName"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(spec.Type, oldSpec.Type, fldPath.Child("type"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(spec.OptionValues, oldSpec.OptionValues, fldPath.Child("optionValues"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(spec.Substitutions, oldSpec.Substitutions, fldPath.Child("substitutions"))...)
	allErrs = append(allErrs, v.ValidateJobTemplateSpecImmutable(oldSpec.Template, spec.Template, fldPath.Child("template"))...)
	allErrs = append(allErrs, v.ValidateKillTimestampUpdate(oldSpec.KillTimestamp, spec.KillTimestamp, fldPath.Child("killTimestamp"))...)
	return allErrs
}

// ValidateJobTemplateSpecImmutable validates that fields in a Job's *v1alpha1.JobTemplateSpec are immutable.
func (v *Validator) ValidateJobTemplateSpecImmutable(oldTemplate, template *v1alpha1.JobTemplateSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(template.Task, oldTemplate.Task, fldPath.Child("task"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(template.MaxAttempts, oldTemplate.MaxAttempts, fldPath.Child("maxAttempts"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(template.RetryDelaySeconds, oldTemplate.RetryDelaySeconds, fldPath.Child("retryDelaySeconds"))...)
	return allErrs
}

// ValidateJobType validates a v1alpha1.JobType.
func (v *Validator) ValidateJobType(jobType v1alpha1.JobType, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	switch jobType {
	case v1alpha1.JobTypeAdhoc, v1alpha1.JobTypeScheduled:
		break
	case "":
		allErrs = append(allErrs, field.Required(fldPath, ""))
	default:
		validValues := []string{
			string(v1alpha1.JobTypeAdhoc),
			string(v1alpha1.JobTypeScheduled),
		}
		allErrs = append(allErrs, field.NotSupported(fldPath, jobType, validValues))
	}
	return allErrs
}

// ValidateKillTimestampUpdate validates update of a KillTimestamp.
func (v *Validator) ValidateKillTimestampUpdate(oldTimestamp, timestamp *metav1.Time, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	now := metav1.NewTime(Clock.Now())

	// Cannot update KillTimestamp if already passed.
	if !oldTimestamp.IsZero() && oldTimestamp.Before(&now) && !oldTimestamp.Equal(timestamp) {
		allErrs = append(allErrs, field.Invalid(fldPath, timestamp, "field is immutable once passed"))
	}

	return allErrs
}

// ValidateStartPolicySpec validates a *v1alpha1.StartPolicySpec.
func (v *Validator) ValidateStartPolicySpec(spec *v1alpha1.StartPolicySpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec != nil {
		allErrs = append(allErrs, v.ValidateConcurrencyPolicy(spec.ConcurrencyPolicy, fldPath.Child("concurrencyPolicy"))...)
	}
	return allErrs
}

// ValidateJobTemplateSpec validates a *v1alpha1.JobTemplateSpec.
func (v *Validator) ValidateJobTemplateSpec(template *v1alpha1.JobTemplateSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, v.ValidateJobTaskSpec(&template.Task, fldPath.Child("task"))...)
	if template.MaxAttempts != nil {
		allErrs = append(allErrs, v.ValidateMaxRetryAttempts(*template.MaxAttempts, fldPath.Child("maxAttempts"))...)
	}
	if template.RetryDelaySeconds != nil {
		allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(*template.RetryDelaySeconds, fldPath.Child("retryDelaySeconds"))...)
	}
	return allErrs
}

func (v *Validator) ValidateMaxRetryAttempts(attempts int32, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Must be greater than 0.
	allErrs = append(allErrs, validation.ValidateGT(int64(attempts), 0, fldPath)...)

	// Set an arbitrary limit of 50 attempts.
	// TODO(irvinlim): Support configuring this value
	allErrs = append(allErrs, validation.ValidateLTE(int64(attempts), 50, fldPath)...)

	return allErrs
}

// ValidateJobTaskSpec validates a *v1alpha1.TaskSpec.
func (v *Validator) ValidateJobTaskSpec(spec *v1alpha1.TaskSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, v.ValidateTaskTemplate(&spec.Template, fldPath.Child("template"))...)
	if spec.PendingTimeoutSeconds != nil {
		allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(*spec.PendingTimeoutSeconds, fldPath.Child("pendingTimeoutSeconds"))...)
	}
	return allErrs
}

// ValidateTaskTemplate validates a *v1alpha1.TaskTemplate.
func (v *Validator) ValidateTaskTemplate(spec *v1alpha1.TaskTemplate, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Exactly one template must be specified.
	var numSpecified int
	if spec.Pod != nil {
		fldPath := fldPath.Child("pod")
		if numSpecified >= 1 {
			allErrs = append(allErrs, field.Forbidden(fldPath, errMessageTooManyTemplates))
		} else {
			numSpecified++
			allErrs = append(allErrs, v.ValidatePodTaskTemplateSpec(spec.Pod, fldPath)...)
		}
	}
	if numSpecified == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "must specify a template type"))
	}

	return allErrs
}

// ValidatePodTaskTemplateSpec validates a *v1alpha1.PodTemplateSpec.
func (v *Validator) ValidatePodTaskTemplateSpec(spec *v1alpha1.PodTemplateSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Convert to corev1.PodTemplateSpec.
	podTemplateSpec := &corev1.PodTemplateSpec{
		ObjectMeta: spec.ObjectMeta,
		Spec:       spec.Spec,
	}

	// Invoke the Kubernetes core validator for the PodTemplateSpec here.
	//
	// Note that even though we do not embed the schema for PodTemplateSpec into our
	// CRD, we should still validate here to catch any invalid PodSpec during
	// creation of the JobConfig/Job, instead of when the Job is about to create the
	// task. We also assume that the Kubernetes core API will not break backwards
	// compatibility, and if we happen to invoke an older version of the Kubernetes
	// core validator against a newer API, it SHOULD NOT break (i.e. cannot save the
	// JobConfig). If a newer validation rule was added in a newer K8s version, then
	// unfortunately we cannot catch it in this scenario, and we will fall back to
	// failing with AdmissionRefusedError only at task creation time.
	allErrs = append(allErrs, ValidatePodTemplateSpec(podTemplateSpec, fldPath)...)

	// Cannot use Always for restartPolicy.
	if restartPolicy := spec.Spec.RestartPolicy; restartPolicy == corev1.RestartPolicyAlways {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("spec.restartPolicy"), restartPolicy, "restartPolicy cannot be Always"))
	}

	return allErrs
}

func (v *Validator) getJobConfigLister(namespace string) executionlister.JobConfigNamespaceLister {
	return v.ctrlContext.Informers().Furiko().Execution().V1alpha1().JobConfigs().Lister().
		JobConfigs(namespace)
}
