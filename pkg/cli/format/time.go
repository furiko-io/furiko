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

package format

import (
	"fmt"
	"math"
	"time"

	"github.com/xeonx/timeago"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

const (
	timeFormat = "Mon, 02 Jan 2006 15:04:05 -07:00"
)

var (
	shorthandPeriods = []timeago.FormatPeriod{
		{D: time.Second, One: "1s", Many: "%ds"},
		{D: time.Minute, One: "1m", Many: "%dm"},
		{D: time.Hour, One: "1d", Many: "%dh"},
		{D: timeago.Day, One: "1d", Many: "%dd"},
	}

	shorthandDurationFormatter = timeago.Config{
		FuturePrefix: "in ",
		Periods:      shorthandPeriods,
		Zero:         "0s",
		Max:          math.MaxInt64, // no max
	}

	standardPeriods = []timeago.FormatPeriod{
		{D: time.Second, One: "about a second", Many: "%d seconds"},
		{D: time.Minute, One: "about a minute", Many: "%d minutes"},
		{D: time.Hour, One: "about an hour", Many: "%d hours"},
		{D: timeago.Day, One: "one day", Many: "%d days"},
		{D: timeago.Month, One: "one month", Many: "%d months"},
		{D: timeago.Year, One: "one year", Many: "%d years"},
	}

	standardDurationFormatter = timeago.Config{
		FuturePrefix: "in ",
		Periods:      standardPeriods,
		Zero:         "about a second",
		Max:          math.MaxInt64,
	}

	standardTimeAgoFormatter = timeago.Config{
		PastSuffix:   " ago",
		FuturePrefix: "in ",
		Periods:      standardPeriods,
		Zero:         "about a second",
		Max:          math.MaxInt64,
	}
)

// TimeFormatFunc is a function that formats a given *metav1.Time into a string format.
type TimeFormatFunc func(t *metav1.Time) string

// TimeAgo formats a time as a string representing the duration since the
// given time.
func TimeAgo(t *metav1.Time) string {
	if t.IsZero() {
		return ""
	}
	return shorthandDurationFormatter.FormatReference(t.Time, ktime.Now().Time)
}

// Duration formats a duration.
func Duration(d time.Duration) string {
	return standardDurationFormatter.FormatRelativeDuration(d)
}

// Time formats a time using a human-readable format.
func Time(t *metav1.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(timeFormat)
}

// TimeWithTimeAgo formats a time together with a relative time ago as a string.
func TimeWithTimeAgo(t *metav1.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%v (%v)", Time(t), standardTimeAgoFormatter.FormatReference(t.Time, ktime.Now().Time))
}
