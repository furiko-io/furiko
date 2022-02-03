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
	configv1 "github.com/furiko-io/furiko/apis/config/v1"
)

var (
	DefaultJobControllerConfig = &configv1.JobControllerConfig{
		DefaultTTLSecondsAfterFinished:        3600,
		DefaultPendingTimeoutSeconds:          900,
		DeleteKillingTasksTimeoutSeconds:      180,
		ForceDeleteKillingTasksTimeoutSeconds: 120,
	}

	DefaultCronControllerConfig = &configv1.CronControllerConfig{
		CronFormat:                  "standard",
		CronHashNames:               mkboolp(true),
		CronHashSecondsByDefault:    mkboolp(false),
		CronHashFields:              mkboolp(true),
		MaxMissedSchedules:          5,
		MaxDowntimeThresholdSeconds: 300,
		DefaultTimezone:             mkstrp("UTC"),
	}
)

func mkstrp(s string) *string {
	return &s
}

func mkboolp(b bool) *bool {
	return &b
}
