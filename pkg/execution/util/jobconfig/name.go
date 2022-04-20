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
	"fmt"
	"time"
)

// GenerateName generates a unique name for a Job.
// Optionally pass a start time to make execution ID generation deterministic.
func GenerateName(jobConfigName string, startTime time.Time) string {
	// If timestamp not specified, default to current time.
	// We recommend to pass a deterministic start time if possible, which helps provide a unique identifier
	// for a single scheduled execution.
	if startTime.IsZero() {
		startTime = time.Now()
	}

	// Encode the start timestamp (up to a second).
	ts := startTime.Unix()

	// Example name:
	// job-sample.1606987620
	return fmt.Sprintf("%v.%v", jobConfigName, ts)
}
