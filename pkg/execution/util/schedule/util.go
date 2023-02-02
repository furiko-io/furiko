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

package schedule

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// IsAllowed returns true if the given schedule is allowed to be triggered on the given time.
func IsAllowed(schedule *execution.ScheduleSpec, t time.Time) bool {
	refTime := metav1.NewTime(t)

	// Schedule is disabled.
	if schedule.Disabled {
		return false
	}

	// Schedule constraints are in place.
	if constraints := schedule.Constraints; constraints != nil {
		if !constraints.NotBefore.IsZero() && refTime.Before(constraints.NotBefore) {
			return false
		}
		if !constraints.NotAfter.IsZero() && refTime.After(constraints.NotAfter.Time) {
			return false
		}
	}

	return true
}
