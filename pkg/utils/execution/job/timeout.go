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

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// GetPendingTimeout returns the pending timeout for the given Job.
func GetPendingTimeout(rj *execution.Job, defaultTimeoutSeconds int64) time.Duration {
	seconds := defaultTimeoutSeconds

	// Override with configured timeout from task.
	if taskPt := rj.Spec.Template.Task.PendingTimeoutSeconds; taskPt != nil && *taskPt >= 0 {
		seconds = *taskPt
	}

	return time.Duration(seconds) * time.Second
}
