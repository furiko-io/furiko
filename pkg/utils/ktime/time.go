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

package ktime

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
)

var (
	// Clock is used to mock time.Now() for tests.
	// Use Clock.Now() instead of time.Now() whenever needed.
	Clock clock.PassiveClock = clock.RealClock{}
)

// Now returns a *metav1.Time pointer for the current time.
func Now() *metav1.Time {
	mtime := metav1.NewTime(Clock.Now())
	return &mtime
}

// IsTimeSetAndEarlier returns true if the given *metav1.Time is set and earlier than the current time.
// Useful to validate timestamp updates which cannot be extended.
func IsTimeSetAndEarlier(ts *metav1.Time) bool {
	return IsTimeSetAndEarlierThan(ts, Clock.Now())
}

// IsTimeSetAndEarlierOrEqual returns true if the given *metav1.Time is set and
// earlier than or equal to the current time. Useful to validate timestamp
// updates which cannot be extended.
func IsTimeSetAndEarlierOrEqual(ts *metav1.Time) bool {
	return IsTimeSetAndEarlierThanOrEqualTo(ts, Clock.Now())
}

// IsTimeSetAndEarlierThan returns true if the given *metav1.Time is set and earlier than the other time.Time.
// Useful to validate timestamp updates which cannot be extended.
func IsTimeSetAndEarlierThan(ts *metav1.Time, other time.Time) bool {
	t := UnwrapMetaV1Time(ts)
	return !t.IsZero() && t.Before(other)
}

// IsTimeSetAndEarlierThanOrEqualTo returns true if the given *metav1.Time is set and earlier than or equal to
// the other time.Time. Useful to validate timestamp updates which cannot be extended.
func IsTimeSetAndEarlierThanOrEqualTo(ts *metav1.Time, other time.Time) bool {
	t := UnwrapMetaV1Time(ts)
	return !t.IsZero() && (t.Before(other) || t.Equal(other))
}

// IsTimeSetAndLater returns true if the given *metav1.Time is set and later than the current time.
// Useful to validate timestamp updates which cannot be extended.
func IsTimeSetAndLater(ts *metav1.Time) bool {
	return IsTimeSetAndLaterThan(ts, Clock.Now())
}

// IsTimeSetAndLaterThan returns true if the given *metav1.Time is set and later than the other time.Time.
// Useful to validate timestamp updates which cannot be extended.
func IsTimeSetAndLaterThan(ts *metav1.Time, other time.Time) bool {
	t := UnwrapMetaV1Time(ts)
	return !t.IsZero() && t.After(other)
}

// IsTimeSetAndLaterThanOrEqualTo returns true if the given *metav1.Time is set and later than or equal to
// the other time.Time. Useful to validate timestamp updates which cannot be extended.
func IsTimeSetAndLaterThanOrEqualTo(ts *metav1.Time, other time.Time) bool {
	t := UnwrapMetaV1Time(ts)
	return !t.IsZero() && (t.After(other) || t.Equal(other))
}

// TimeMax returns the maximum of two *metav1.Time objects.
func TimeMax(ts1, ts2 *metav1.Time) *metav1.Time {
	if ts1 == nil {
		return ts2
	}
	if ts2 == nil {
		return ts1
	}
	if ts1.Before(ts2) {
		return ts2
	}
	return ts1
}

// UnwrapMetaV1Time unwraps a *metav1.Time pointer into time.Time.
// If nil, a zero time is returned.
func UnwrapMetaV1Time(ts *metav1.Time) time.Time {
	if ts != nil {
		return ts.Time
	}
	return time.Time{}
}

// IsUnixZero provides a more robust check for whether a metav1.Time is truly a
// zero time, by also checking if it is equivalent to the epoch (i.e. Unix()
// returns 0).
func IsUnixZero(ts *metav1.Time) bool {
	return ts.IsZero() || ts.Unix() == 0
}
