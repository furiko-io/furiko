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
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// IsStarted returns true if the Job is started.
func IsStarted(rj *execution.Job) bool {
	return !rj.Status.StartTime.IsZero()
}

// IsQueued returns true if the Job is queued. A queued job is one that is not
// started and not terminal.
func IsQueued(rj *execution.Job) bool {
	return !IsStarted(rj) && !rj.Status.Phase.IsTerminal()
}

// IsActive returns true if the Job is considered active. An active job is one
// that is started and not terminal.
func IsActive(rj *execution.Job) bool {
	return IsStarted(rj) && !rj.Status.Phase.IsTerminal()
}
