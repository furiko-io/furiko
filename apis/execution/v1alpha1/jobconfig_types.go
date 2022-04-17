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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// JobConfigSpec defines the desired state of the JobConfig.
type JobConfigSpec struct {
	// Template for creating the Job.
	Template JobTemplate `json:"template"`

	// Concurrency defines the behaviour of multiple concurrent Jobs.
	Concurrency ConcurrencySpec `json:"concurrency"`

	// Schedule is an optional field that defines automatic scheduling of the
	// JobConfig.
	//
	// +optional
	Schedule *ScheduleSpec `json:"schedule,omitempty"`

	// Option is an optional field that defines how the JobConfig is parameterized.
	// Each option defined here can subsequently be used in the Template via context
	// variable substitution.
	//
	// +optional
	Option *OptionSpec `json:"option,omitempty"`
}

// ConcurrencySpec defines how to handle multiple concurrent Jobs for the JobConfig.
type ConcurrencySpec struct {
	// Policy describes how to treat concurrent executions of the same JobConfig.
	Policy ConcurrencyPolicy `json:"policy"`
}

// ScheduleSpec defines how a JobConfig should be automatically scheduled.
type ScheduleSpec struct {
	// Specify a schedule using cron expressions.
	//
	// +optional
	Cron *CronSchedule `json:"cron,omitempty"`

	// If true, then automatic scheduling will be disabled for the JobConfig.
	// +optional
	Disabled bool `json:"disabled"`

	// Specifies any constraints that should apply to this Schedule.
	//
	// +optional
	Constraints *ScheduleContraints `json:"constraints,omitempty"`

	// Specifies the time that the schedule was last upated. This prevents
	// accidental back-scheduling.
	//
	// For example, if a JobConfig that was previously disabled from automatic
	// scheduling is now enabled, we do not want to perform back-scheduling for
	// schedules after LastScheduled prior to updating of the JobConfig.
	//
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// CronSchedule defines a Schedule based on cron format.
type CronSchedule struct {
	// Cron expression to specify how the JobConfig will be periodically scheduled.
	// Example: "0 0/5 * * *".
	//
	// Supports cron schedules with optional "seconds" and "years" fields, i.e. can
	// parse between 5 to 7 tokens.
	//
	// More information: https://github.com/furiko-io/cronexpr
	Expression string `json:"expression"`

	// Timezone to interpret the cron schedule in. For example, a cron schedule of
	// "0 10 * * *" with a timezone of "Asia/Singapore" will be interpreted as
	// running at 02:00:00 UTC time every day.
	//
	// Timezone must be one of the following:
	//
	//  1. A valid tz string (e.g. "Asia/Singapore", "America/New_York").
	//  2. A UTC offset with minutes (e.g. UTC-10:00).
	//  3. A GMT offset with minutes (e.g. GMT+05:30). The meaning is the
	//     same as its UTC counterpart.
	//
	// This field merely is used for parsing the cron Expression, and has nothing to
	// do with /etc/timezone inside the container (i.e. it will not set $TZ
	// automatically).
	//
	// Defaults to the controller's default configured timezone.
	//
	// +optional
	Timezone string `json:"timezone,omitempty"`
}

// ScheduleContraints defines constraints for automatic scheduling.
type ScheduleContraints struct {
	// Specifies the earliest possible time that is allowed to be scheduled. If set,
	// the scheduler should not create schedules before this time.
	//
	// +optional
	NotBefore *metav1.Time `json:"notBefore,omitempty"`

	// Specifies the latest possible time that is allowed to be scheduled. If set,
	// the scheduler should not create schedules after this time.
	//
	// +optional
	NotAfter *metav1.Time `json:"notAfter,omitempty"`
}

type ConcurrencyPolicy string

const (
	// ConcurrencyPolicyAllow allows multiple Job objects for a single JobConfig to
	// be created concurrently.
	ConcurrencyPolicyAllow ConcurrencyPolicy = "Allow"

	// ConcurrencyPolicyForbid forbids multiple Job objects for a single JobConfig
	// to be created concurrently. If a new Job is set to be automatically scheduled
	// with another concurrent Job, the new Job will be dropped from the queue.
	ConcurrencyPolicyForbid ConcurrencyPolicy = "Forbid"

	// ConcurrencyPolicyEnqueue enqueues the Job to be run after other Jobs for the
	// JobConfig have finished. If a new Job is set to be automatically scheduld
	// with another concurrent Job, the new Job will wait until the previous Job
	// finishes.
	ConcurrencyPolicyEnqueue ConcurrencyPolicy = "Enqueue"
)

// OptionSpec defines how a JobConfig is parameterized using Job Options.
type OptionSpec struct {
	// Options is a list of job options.
	// +optional
	Options []Option `json:"options,omitempty"`
}

// Option defines a single job option.
type Option struct {
	// The type of the job option.
	// Can be one of: bool, string, select, multi, date
	Type OptionType `json:"type"`

	// The name of the job option. Will be substituted as `${option.NAME}`.
	// Must match the following regex: ^[a-zA-Z_0-9.-]+$
	Name string `json:"name"`

	// Label is an optional human-readable label for this option, which is purely
	// used for display purposes.
	//
	// +optional
	Label string `json:"label,omitempty"`

	// Required defines whether this field is required.
	//
	// Default: false
	// +optional
	Required bool `json:"required,omitempty"`

	// Bool adds additional configuration for OptionTypeBool.
	// +optional
	Bool *BoolOptionConfig `json:"bool,omitempty"`

	// String adds additional configuration for OptionTypeString.
	// +optional
	String *StringOptionConfig `json:"string,omitempty"`

	// Select adds additional configuration for OptionTypeSelect.
	// +optional
	Select *SelectOptionConfig `json:"select,omitempty"`

	// Multi adds additional configuration for OptionTypeMulti.
	// +optional
	Multi *MultiOptionConfig `json:"multi,omitempty"`

	// Date adds additional configuration for OptionTypeDate.
	// +optional
	Date *DateOptionConfig `json:"date,omitempty"`
}

type OptionType string

const (
	// OptionTypeBool defines an Option whose type is a boolean value.
	OptionTypeBool OptionType = "Bool"

	// OptionTypeString defines an Option whose type is an arbitrary string.
	OptionTypeString OptionType = "String"

	// OptionTypeSelect defines an Option whose value comes from a list of strings.
	OptionTypeSelect OptionType = "Select"

	// OptionTypeMulti defines an Option that contains zero or more values from a
	// list of strings, delimited with some delimiter.
	OptionTypeMulti OptionType = "Multi"

	// OptionTypeDate defines an Option whose value is a formatted date string.
	OptionTypeDate OptionType = "Date"
)

var OptionTypesAll = []OptionType{
	OptionTypeBool,
	OptionTypeString,
	OptionTypeSelect,
	OptionTypeMulti,
	OptionTypeDate,
}

func (t OptionType) IsValid() bool {
	for _, o := range OptionTypesAll {
		if o == t {
			return true
		}
	}
	return false
}

// BoolOptionConfig defines the options for OptionTypeBool.
type BoolOptionConfig struct {
	// Default value, will be used to populate the option if not specified.
	Default bool `json:"default"`

	// Determines how to format the value as string.
	// Can be one of: TrueFalse, OneZero, YesNo, Custom
	//
	// Default: TrueFalse
	// +optional
	Format BoolOptionFormat `json:"format,omitempty"`

	// If Format is custom, will be substituted if value is true.
	// Can also be an empty string.
	//
	// +optional
	TrueVal string `json:"trueVal,omitempty"`

	// If Format is custom, will be substituted if value is false.
	// Can also be an empty string.
	//
	// +optional
	FalseVal string `json:"falseVal,omitempty"`
}

// BoolOptionFormat describes how a bool option should be formatted.
type BoolOptionFormat string

const (
	// BoolOptionFormatTrueFalse formats bool values as "true" and "false"
	// respectively.
	BoolOptionFormatTrueFalse BoolOptionFormat = "TrueFalse"

	// BoolOptionFormatOneZero formats bool values as "1" and "0" respectively.
	BoolOptionFormatOneZero BoolOptionFormat = "OneZero"

	// BoolOptionFormatYesNo formats bool values as "yes" and "no" respectively.
	BoolOptionFormatYesNo BoolOptionFormat = "YesNo"

	// BoolOptionFormatCustom formats bool values according to a custom format.
	BoolOptionFormatCustom BoolOptionFormat = "Custom"
)

var BoolOptionFormatsAll = []BoolOptionFormat{
	BoolOptionFormatTrueFalse,
	BoolOptionFormatOneZero,
	BoolOptionFormatYesNo,
	BoolOptionFormatCustom,
}

var boolOptionFormatStrings = map[BoolOptionFormat]map[bool]string{
	BoolOptionFormatTrueFalse: {
		true:  "true",
		false: "false",
	},
	BoolOptionFormatOneZero: {
		true:  "1",
		false: "0",
	},
	BoolOptionFormatYesNo: {
		true:  "yes",
		false: "no",
	},
}

func (b BoolOptionFormat) Format(val bool) (string, error) {
	formats, ok := boolOptionFormatStrings[b]
	if !ok {
		return "", fmt.Errorf("invalid format: %v", b)
	}
	return formats[val], nil
}

func (b BoolOptionFormat) IsValid() bool {
	for _, f := range BoolOptionFormatsAll {
		if b == f {
			return true
		}
	}
	return false
}

func (c *BoolOptionConfig) FormatValue(value bool) (string, error) {
	if c.Format == BoolOptionFormatCustom {
		if value {
			return c.TrueVal, nil
		}
		return c.FalseVal, nil
	}
	return c.Format.Format(value)
}

// StringOptionConfig defines the options for OptionTypeString.
type StringOptionConfig struct {
	// Optional default value, will be used to populate the option if not specified.
	// +optional
	Default string `json:"default,omitempty"`

	// Whether to trim spaces before substitution.
	//
	// Default: false
	// +optional
	TrimSpaces bool `json:"trimSpaces,omitempty"`
}

// SelectOptionConfig defines the options for OptionTypeSelect.
type SelectOptionConfig struct {
	// Default value, will be used to populate the option if not specified.
	// +optional
	Default string `json:"default,omitempty"`

	// List of values to be chosen from.
	// +optional
	Values []string `json:"values"`

	// Whether to allow custom values instead of just the list of allowed values.
	//
	// Default: false
	// +optional
	AllowCustom bool `json:"allowCustom,omitempty"`
}

// MultiOptionConfig defines the options for OptionTypeMulti.
type MultiOptionConfig struct {
	// Default values, will be used to populate the option if not specified.
	// +optional
	Default []string `json:"default,omitempty"`

	// Delimiter to join values by.
	Delimiter string `json:"delimiter"`

	// List of values to be chosen from.
	Values []string `json:"values"`

	// Whether to allow custom values instead of just the list of allowed values.
	//
	// Default: false
	// +optional
	AllowCustom bool `json:"allowCustom,omitempty"`
}

// DateOptionConfig defines the options for OptionTypeDate.
type DateOptionConfig struct {
	// Date format in moment.js format. If not specified, will use RFC3339 format by
	// default.
	//
	// Date format reference: https://momentjs.com/docs/#/displaying/format/
	//
	// Default:
	// +optional
	Format string `json:"format,omitempty"`
}

// JobConfigStatus defines the observed state of the JobConfig.
type JobConfigStatus struct {
	// Human-readable and high-level representation of the status of the JobConfig.
	State JobConfigState `json:"state"`

	// Total number of Jobs queued for the JobConfig. A job that is queued is one
	// that is not yet started.
	//
	// +optional
	Queued int64 `json:"queued"`

	// A list of pointers to Job objects queued for the JobConfig.
	//
	// +optional
	QueuedJobs []JobReference `json:"queuedJobs,omitempty"`

	// Total number of active jobs created for the JobConfig. An active job is one
	// that is waiting to create a task, waiting for a task to be running, or has a
	// running task.
	//
	// +optional
	Active int64 `json:"active"`

	// A list of pointers to active Job objects for the JobConfig.
	//
	// +optional
	ActiveJobs []JobReference `json:"activeJobs,omitempty"`

	// The last known schedule time for this job config, used to persist state
	// during controller downtime. If the controller was down for a short period of
	// time, any schedules that were missed during the downtime will be
	// back-scheduled, subject to the number of schedules missed since
	// LastScheduled.
	//
	// +optional
	LastScheduled *metav1.Time `json:"lastScheduled,omitempty"`
}

type JobConfigState string

const (
	// JobConfigReady means that the JobConfig is ready to be executed. This state
	// is used if the JobConfig has no schedule set, otherwise a more specific state
	// like ReadyDisabled or ReadyEnabled is preferred.
	JobConfigReady JobConfigState = "Ready"

	// JobConfigReadyEnabled means that the JobConfig has a schedule specified
	// which is enabled, and is ready to be executed.
	JobConfigReadyEnabled JobConfigState = "ReadyEnabled"

	// JobConfigReadyDisabled means that the JobConfig has a schedule specified
	// which is disabled, and is ready to be executed.
	JobConfigReadyDisabled JobConfigState = "ReadyDisabled"

	// JobConfigJobQueued means that the JobConfig has some Job(s) that are Queued,
	// and none of them are started yet.
	JobConfigJobQueued JobConfigState = "JobQueued"

	// JobConfigExecuting means that the JobConfig has some Job already started and
	// may be running.
	JobConfigExecuting JobConfigState = "Executing"
)

type JobReference struct {
	// UID of the Job.
	UID types.UID `json:"uid"`

	// Name of the Job.
	Name string `json:"name"`

	// Timestamp that the Job was created.
	CreationTimestamp metav1.Time `json:"creationTimestamp"`

	// Timestamp that the Job was started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
}

// nolint:lll
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=furikojobconfig;furikojobconfigs
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Active",type=string,JSONPath=`.status.active`
// +kubebuilder:printcolumn:name="Queued",type=string,JSONPath=`.status.queued`
// +kubebuilder:printcolumn:name="Cron Schedule",type=string,JSONPath=`.spec.schedule.cron.expression`
// +kubebuilder:printcolumn:name="Timezone",type=string,JSONPath=`.spec.schedule.cron.timezone`
// +kubebuilder:printcolumn:name="Last Schedule Time",type=date,JSONPath=`.status.lastScheduleTime`
// +kubebuilder:webhook:path=/mutating/jobconfigs.execution.furiko.io,mutating=true,failurePolicy=fail,sideEffects=None,groups=execution.furiko.io,resources=jobconfigs,verbs=create;update,versions=*,name=mutating.webhook.jobconfigs.execution.furiko.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validating/jobconfigs.execution.furiko.io,mutating=false,failurePolicy=fail,sideEffects=None,groups=execution.furiko.io,resources=jobconfigs,verbs=create;update,versions=*,name=validation.webhook.jobconfigs.execution.furiko.io,admissionReviewVersions=v1

// JobConfig is the schema for a single job configuration. Multiple Job objects
// belong to a single JobConfig.
type JobConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobConfigSpec   `json:"spec,omitempty"`
	Status JobConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobConfigList contains a list of JobConfig objects.
type JobConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JobConfig{}, &JobConfigList{})
}
