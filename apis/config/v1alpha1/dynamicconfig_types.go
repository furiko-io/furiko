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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobExecutionConfig defines runtime global configuration for Jobs in the cluster.
type JobExecutionConfig struct {
	metav1.TypeMeta `json:",inline"`

	// DefaultTTLSecondsAfterFinished is the default time-to-live (TTL) for a Job
	// after it has finished. Lower this value to reduce the strain on the
	// cluster/kubelet. Set to 0 to delete immediately after the Job is finished.
	//
	// Default: 3600
	// +optional
	DefaultTTLSecondsAfterFinished *int64 `json:"defaultTTLSecondsAfterFinished,omitempty"`

	// DefaultPendingTimeoutSeconds is default timeout to use if job does not
	// specify the pending timeout. By default, this is a non-zero value to prevent
	// permanently stuck jobs. To disable default pending timeout, set this to 0.
	//
	// Default: 900
	// +optional
	DefaultPendingTimeoutSeconds *int64 `json:"defaultPendingTimeoutSeconds,omitempty"`

	// DeleteKillingTasksTimeoutSeconds is the duration we delete the task to kill
	// it instead of using active deadline, if previous efforts were ineffective.
	// Set this value to 0 to immediately use deletion.
	//
	// Default: 180
	// +optional
	DeleteKillingTasksTimeoutSeconds *int64 `json:"deleteKillingTasksTimeoutSeconds,omitempty"`

	// ForceDeleteKillingTasksTimeoutSeconds is the duration before we use force
	// deletion instead of normal deletion. This timeout is computed from the
	// deletionTimestamp of the object, which may also include an additional delay
	// of deletionGracePeriodSeconds. Set this value to 0 to disable force deletion.
	//
	// Default: 120
	// +optional
	ForceDeleteKillingTasksTimeoutSeconds *int64 `json:"forceDeleteKillingTasksTimeoutSeconds,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobConfigExecutionConfig defines runtime global configuration for JobConfigs in the cluster.
type JobConfigExecutionConfig struct {
	metav1.TypeMeta `json:",inline"`

	// MaxEnqueuedJobs is the global maximum enqueued jobs that can be enqueued for
	// a single JobConfig.
	//
	// Default: 20
	// +optional
	MaxEnqueuedJobs *int64 `json:"maxEnqueuedJobs,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronExecutionConfig defines runtime global configuration for Cron in the cluster.
type CronExecutionConfig struct {
	metav1.TypeMeta `json:",inline"`

	// CronFormat specifies the format used to parse cron expressions. Select
	// between "standard" (default) or "quartz". More info can be found at
	// https://github.com/furiko-io/cronexpr.
	//
	// Default: standard
	// +optional
	CronFormat string `json:"cronFormat,omitempty"`

	// CronHashNames specifies if cron expressions should be hashed using the
	// JobConfig's name.
	//
	// This enables "hash cron expressions", which looks like `0 H * * *`. This
	// particular example means to run once a day on the 0th minute of some hour,
	// which will be determined by hashing the JobConfig's name. By enabling this
	// option, JobConfigs that use such cron schedules will be load balanced across
	// the cluster.
	//
	// If disabled, any JobConfigs that use the `H` syntax will throw a parse error.
	//
	// Default: true
	// +optional
	CronHashNames *bool `json:"cronHashNames,omitempty"`

	// CronHashSecondsByDefault specifies if the seconds field of a cron expression
	// should be a `H` or `0` by default. If enabled, it will be `H`, otherwise it
	// will default to `0`.
	//
	// For JobConfigs which use a short cron expression format (i.e. 5 or 6 tokens
	// long), the seconds field is omitted and is typically assumed to be `0` (e.g.
	// `5 10 * * *` means to run at 10:05:00 every day). Enabling this option will
	// allow JobConfigs to be scheduled across the minute, improving load balancing.
	//
	// Users can still choose to start at 0 seconds by explicitly specifying a long
	// cron expression format with `0` in the seconds field. In the above example,
	// this would be `0 5 10 * * * *`.
	//
	// Default: false
	// +optional
	CronHashSecondsByDefault *bool `json:"cronHashSecondsByDefault,omitempty"`

	// CronHashFields specifies if the fields should be hashed along with the
	// JobConfig's name.
	//
	// For example, `H H * * * * *` will always hash the seconds and minutes to the
	// same value, for example 00:37:37, 01:37:37, etc. Enabling this option will
	// append additional keys to be hashed to introduce additional non-determinism.
	//
	// Default: true
	// +optional
	CronHashFields *bool `json:"cronHashFields,omitempty"`

	// DefaultTimezone defines a default timezone to use for JobConfigs that do not
	// specify a timezone. If left empty, UTC will be used as the default timezone.
	//
	// Default: UTC
	// +optional
	DefaultTimezone *string `json:"defaultTimezone,omitempty"`

	// MaxMissedSchedules defines a maximum number of jobs that the controller
	// should back-schedule, or attempt to create after coming back up from
	// downtime. Having a sane value here would prevent a thundering herd of jobs
	// being scheduled that would exhaust resources in the cluster. Set this to 0 to
	// disable back-scheduling.
	//
	// Default: 5
	// +optional
	MaxMissedSchedules *int64 `json:"maxMissedSchedules,omitempty"`

	// MaxDowntimeThresholdSeconds defines the maximum downtime that the controller
	// can tolerate. If the controller was intentionally shut down for an extended
	// period of time, we should not attempt to back-schedule jobs once it was
	// started.
	//
	// Default: 300
	// +optional
	MaxDowntimeThresholdSeconds int64 `json:"maxDowntimeThresholdSeconds,omitempty"`
}

func init() {
	SchemeBuilder.Register(&JobExecutionConfig{}, &CronExecutionConfig{})
}
