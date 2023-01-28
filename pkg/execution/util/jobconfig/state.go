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

package jobconfig

import (
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// GetState returns the high-level state of a JobConfig.
func GetState(rjc *execution.JobConfig) execution.JobConfigState {
	if rjc.Status.Active > 0 {
		return execution.JobConfigExecuting
	}

	if rjc.Status.Queued > 0 {
		return execution.JobConfigJobQueued
	}

	if spec := rjc.Spec.Schedule; spec != nil && spec.Cron != nil {
		if spec.Disabled {
			return execution.JobConfigReadyDisabled
		}
		return execution.JobConfigReadyEnabled
	}

	return execution.JobConfigReady
}

// IsScheduleEnabled returns true if the JobConfig has a schedule and is enabled.
func IsScheduleEnabled(jobConfig *execution.JobConfig) bool {
	scheduleSpec := jobConfig.Spec.Schedule
	return scheduleSpec != nil && !scheduleSpec.Disabled
}
