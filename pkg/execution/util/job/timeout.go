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

package job

import (
	"time"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// GetPendingTimeout returns the pending timeout for the given Job.
func GetPendingTimeout(rj *execution.Job, cfg *configv1alpha1.JobExecutionConfig) time.Duration {
	var sec int64
	if spec := cfg.DefaultPendingTimeoutSeconds; spec != nil {
		sec = *spec
	}
	if taskPt := rj.Spec.Template.TaskPendingTimeoutSeconds; taskPt != nil && *taskPt >= 0 {
		sec = *taskPt
	}
	return time.Duration(sec) * time.Second
}

// GetForceDeleteTimeout returns the timeout before the controller starts force deletion.
func GetForceDeleteTimeout(cfg *configv1alpha1.JobExecutionConfig) time.Duration {
	var sec int64
	if spec := cfg.ForceDeleteTaskTimeoutSeconds; spec != nil {
		sec = *spec
	}
	return time.Duration(sec) * time.Second
}

// GetTTLAfterFinished returns the TTL after a Job is finished.
func GetTTLAfterFinished(rj *execution.Job, cfg *configv1alpha1.JobExecutionConfig) time.Duration {
	var sec int64
	if spec := cfg.DefaultTTLSecondsAfterFinished; spec != nil {
		sec = *spec
	}
	if ttl := rj.Spec.TTLSecondsAfterFinished; ttl != nil {
		sec = *ttl
	}
	return time.Duration(sec) * time.Second
}
