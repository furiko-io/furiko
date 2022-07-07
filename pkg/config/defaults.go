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

package config

import (
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
)

const (
	// DefaultCronTimezone is the default timezone value that will be used if there
	// is no timezone configuration for the JobConfig or a default value set for the
	// controller.
	DefaultCronTimezone = "UTC"

	// DefaultCronMaxDowntimeThresholdSeconds is the default maximum downtime
	// threshold (in seconds) that the cron controller can tolerate to automatically
	// back-schedule missed schedules, if a value is not explicitly set.
	DefaultCronMaxDowntimeThresholdSeconds = 300

	// DefaultCronMaxMissedSchedules is the default maximum number of schedules that
	// the controller should attempt to back-schedule, if a value is not explicitly
	// set.
	DefaultCronMaxMissedSchedules = 5
)

var (
	DefaultJobExecutionConfig = &configv1alpha1.JobExecutionConfig{
		DefaultTTLSecondsAfterFinished: pointer.Int64(3600),
		DefaultPendingTimeoutSeconds:   pointer.Int64(900),
		ForceDeleteTaskTimeoutSeconds:  pointer.Int64(900),
	}

	DefaultJobConfigExecutionConfig = &configv1alpha1.JobConfigExecutionConfig{
		MaxEnqueuedJobs: pointer.Int64(20),
	}

	DefaultCronExecutionConfig = &configv1alpha1.CronExecutionConfig{
		CronFormat:                  "standard",
		CronHashNames:               pointer.Bool(true),
		CronHashSecondsByDefault:    pointer.Bool(false),
		CronHashFields:              pointer.Bool(true),
		MaxMissedSchedules:          pointer.Int64(DefaultCronMaxMissedSchedules),
		MaxDowntimeThresholdSeconds: DefaultCronMaxDowntimeThresholdSeconds,
		DefaultTimezone:             pointer.String(DefaultCronTimezone),
	}
)
