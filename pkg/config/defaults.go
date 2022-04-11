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

var (
	DefaultJobExecutionConfig = &configv1alpha1.JobExecutionConfig{
		DefaultTTLSecondsAfterFinished:        pointer.Int64(3600),
		DefaultPendingTimeoutSeconds:          pointer.Int64(900),
		DeleteKillingTasksTimeoutSeconds:      pointer.Int64(180),
		ForceDeleteKillingTasksTimeoutSeconds: pointer.Int64(120),
	}

	DefaultCronExecutionConfig = &configv1alpha1.CronExecutionConfig{
		CronFormat:                  "standard",
		CronHashNames:               pointer.Bool(true),
		CronHashSecondsByDefault:    pointer.Bool(false),
		CronHashFields:              pointer.Bool(true),
		MaxMissedSchedules:          pointer.Int64(5),
		MaxDowntimeThresholdSeconds: 300,
		DefaultTimezone:             pointer.String("UTC"),
	}
)
