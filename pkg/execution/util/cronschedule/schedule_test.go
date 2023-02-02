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

package cronschedule_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	fakeclock "k8s.io/utils/clock/testing"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/tzutils"
	"github.com/furiko-io/furiko/pkg/execution/util/cronschedule"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	now = "2021-02-09T04:06:13.234Z"
)

var (
	lastScheduleTime     = testutils.Mkmtime("2021-02-09T04:02:00Z")
	longLastScheduleTime = testutils.Mkmtime("2021-02-09T02:46:00Z")
	notBeforeTime        = testutils.Mkmtime("2021-02-09T04:01:00Z")
	notAfterTime         = testutils.Mkmtime("2021-02-09T04:08:00Z")
	futureTime           = testutils.Mkmtime("2021-02-10T04:02:00Z")

	jobConfigNotScheduled = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-not-scheduled",
		},
		Spec: execution.JobConfigSpec{},
	}

	jobConfigEveryMinute = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-every-minute",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
			},
		},
	}

	jobConfigEveryFiveMinutes = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-every-five-minutes",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "*/5 * * * *",
				},
			},
		},
	}

	jobConfigPointInTime = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-point-in-time",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "0 12 9 2 * 2021",
				},
			},
		},
	}

	jobConfigPointInTimeWithAlreadySetLastScheduleTime = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-point-in-time-with-already-set-last-schedule-time",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "15 6 28 10 * 2021",
				},
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: testutils.Mkmtimep("2021-10-27T22:15:00Z"),
		},
	}

	jobConfigWithLastScheduleTime = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-last-schedule-time",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: &lastScheduleTime,
		},
	}

	jobConfigWithLastScheduleTimeLongAgo = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-last-schedule-time",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: &longLastScheduleTime,
		},
	}

	jobConfigWithTimezone = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-timezone",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "0 0 1/2 1/1 * ? *",
					Timezone:   "Asia/Ho_Chi_Minh",
				},
			},
		},
	}

	jobConfigWithTimezoneAndLastScheduleTime = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-timezone-and-last-schedule-time",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "0 0 1/2 1/1 * ? *",
					Timezone:   "Asia/Ho_Chi_Minh",
				},
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: testutils.Mkmtimep("2021-06-24T09:00:00Z"),
		},
	}

	jobConfigWithLastScheduledAndUpdated = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-last-schedule-and-updated",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				LastUpdated: testutils.Mkmtimep(now),
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: &longLastScheduleTime,
		},
	}

	jobConfigWithNotBeforeConstraint = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-not-before-constraint",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotBefore: &futureTime,
				},
			},
		},
	}

	jobConfigWithNotBeforeAndLastScheduled = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-not-before-and-last-scheduled",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotBefore: &futureTime,
				},
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: &lastScheduleTime,
		},
	}

	jobConfigWithNotBeforeAndLastUpdated = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-not-before-and-last-updated",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotBefore: &notBeforeTime,
				},
				LastUpdated: testutils.Mkmtimep(now),
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: &lastScheduleTime,
		},
	}

	jobConfigWithFutureNotBeforeAndLastUpdated = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-future-not-before-and-last-updated",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotBefore: &futureTime,
				},
				LastUpdated: testutils.Mkmtimep(now),
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: &lastScheduleTime,
		},
	}

	jobConfigWithNotAfterConstraint = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-not-after-constraint",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotAfter: &notAfterTime,
				},
			},
		},
	}

	jobConfigWithNotAfterWithLastScheduled = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-not-after-and-last-scheduled",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotAfter: &notAfterTime,
				},
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: &lastScheduleTime,
		},
	}

	jobConfigWithNotAfterWithLastUpdated = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-not-after-and-last-updated",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotAfter: &notAfterTime,
				},
				LastUpdated: testutils.Mkmtimep(now),
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduled: &lastScheduleTime,
		},
	}

	multiCronJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "multi-cron-job-config",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expressions: []string{
						"0/5 10-18 * * *",
						"0 0-9,19-23 * * *",
					},
				},
			},
		},
	}

	multiCronJobConfigWithOverlap = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "multi-cron-job-config-with-overlap",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expressions: []string{
						"0/5 10-18 * * *", // every 5 minutes from 10-18th hour
						"0 * * * *",       // every hour otherwise, note the overlap with 10:00-18:00 on the 0th minute
						"0/5 10-18 * * *", // repeated expression
					},
				},
			},
		},
	}
)

type JobConfigSchedule struct {
	Time      time.Time
	JobConfig *execution.JobConfig
}

type KeySchedule struct {
	Time time.Time
	Key  string
}

type PopSchedule struct {
	Name string
	At   time.Time
	Want []JobConfigSchedule
}

func TestSchedule(t *testing.T) {
	opts := cmp.Options{
		cmpopts.EquateEmpty(),
		cmpopts.SortSlices(func(i, j KeySchedule) bool {
			if !i.Time.Equal(j.Time) {
				return i.Time.Before(j.Time)
			}
			return i.Key < j.Key
		}),
	}

	tests := []struct {
		name       string
		jobConfigs []*execution.JobConfig
		cfg        *configv1alpha1.CronExecutionConfig
		initTime   time.Time
		cases      []PopSchedule
	}{
		{
			name: "Example scheduling with multiple JobConfigs",
			jobConfigs: []*execution.JobConfig{
				jobConfigNotScheduled,
				jobConfigEveryMinute,
			},
			initTime: testutils.Mktime(now),
			cases: []PopSchedule{
				{
					Name: "No schedules yet immediately after initialization",
					At:   testutils.Mktime(now),
				},
				{
					Name: "Start scheduling after 04:07",
					At:   testutils.Mktime("2021-02-09T04:07:13.234Z"),
					Want: []JobConfigSchedule{
						{Time: testutils.Mktime("2021-02-09T04:07:00Z"), JobConfig: jobConfigEveryMinute},
					},
				},
				{
					Name: "No schedules until 04:08",
					At:   testutils.Mktime("2021-02-09T04:07:25Z"),
				},
				{
					Name: "Missed scheduling at 04:08",
					At:   testutils.Mktime("2021-02-09T04:10:01Z"),
					Want: []JobConfigSchedule{
						{Time: testutils.Mktime("2021-02-09T04:08:00Z"), JobConfig: jobConfigEveryMinute},
					},
				},
				{
					Name: "No schedules until 04:11",
					At:   testutils.Mktime("2021-02-09T04:10:54Z"),
				},
			},
		},
		{
			name: "JobConfig with LastScheduled",
			jobConfigs: []*execution.JobConfig{
				jobConfigWithLastScheduleTime,
			},
			initTime: testutils.Mktime(now),
			cases: []PopSchedule{
				{
					Name: "Pop from LastScheduled",
					At:   testutils.Mktime(now),
					Want: []JobConfigSchedule{
						{Time: testutils.Mktime("2021-02-09T04:03:00Z"), JobConfig: jobConfigWithLastScheduleTime},
					},
				},
				{
					Name: "Bump to next minute",
					At:   testutils.Mktime(now).Add(time.Minute),
					Want: []JobConfigSchedule{
						{Time: testutils.Mktime("2021-02-09T04:07:00Z"), JobConfig: jobConfigWithLastScheduleTime},
					},
				},
			},
		},
		{
			name: "JobConfig with LastScheduled and LastUpdated",
			jobConfigs: []*execution.JobConfig{
				jobConfigWithLastScheduledAndUpdated,
			},
			initTime: testutils.Mktime(now),
			cases: []PopSchedule{
				{
					Name: "No Pop due to LastUpdated",
					At:   testutils.Mktime(now),
				},
				{
					Name: "Bump to next minute",
					At:   testutils.Mktime(now).Add(time.Minute),
					Want: []JobConfigSchedule{
						{Time: testutils.Mktime("2021-02-09T04:07:00Z"), JobConfig: jobConfigWithLastScheduledAndUpdated},
					},
				},
			},
		},
		{
			name: "Multiple JobConfigs",
			jobConfigs: []*execution.JobConfig{
				jobConfigEveryMinute,
				jobConfigEveryFiveMinutes,
			},
			initTime: testutils.Mktime("2021-02-09T04:08:36Z"),
			cases: []PopSchedule{
				{
					Name: "Only schedule jobConfigEveryMinute",
					At:   testutils.Mktime("2021-02-09T04:09:00Z"),
					Want: []JobConfigSchedule{
						{Time: testutils.Mktime("2021-02-09T04:09:00Z"), JobConfig: jobConfigEveryMinute},
					},
				},
				{
					Name: "Schedule both JobConfigs",
					At:   testutils.Mktime("2021-02-09T04:10:00Z"),
					Want: []JobConfigSchedule{
						{Time: testutils.Mktime("2021-02-09T04:10:00Z"), JobConfig: jobConfigEveryMinute},
						{Time: testutils.Mktime("2021-02-09T04:10:00Z"), JobConfig: jobConfigEveryFiveMinutes},
					},
				},
				{
					Name: "Only schedule jobConfigEveryMinute",
					At:   testutils.Mktime("2021-02-09T04:11:00Z"),
					Want: []JobConfigSchedule{
						{Time: testutils.Mktime("2021-02-09T04:11:00Z"), JobConfig: jobConfigEveryMinute},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			initTime := testutils.Mktime(now)
			if !tt.initTime.IsZero() {
				initTime = tt.initTime
			}

			// Initialize the Schedule from initTime.
			fakeClock := fakeclock.NewFakeClock(initTime)
			sched, err := cronschedule.New(
				tt.jobConfigs,
				cronschedule.WithClock(fakeClock),
				cronschedule.WithConfigLoader(&configLoader{cfg: tt.cfg}),
			)
			if err != nil {
				t.Errorf("cannot initialize schedule: %v", err)
				return
			}

			for _, pop := range tt.cases {
				// Pop all schedules until exhausted.
				var popped []KeySchedule
				for {
					key, ts, ok := sched.Pop(pop.At)
					if !ok {
						break
					}
					popped = append(popped, KeySchedule{
						Time: ts,
						Key:  key,
					})
				}

				want := make([]KeySchedule, 0, len(pop.Want))
				for _, s := range pop.Want {
					want = append(want, KeySchedule{
						Time: s.Time,
						Key:  makeKey(s.JobConfig),
					})
				}

				// Ensure that the schedules are equal.
				// We sort the schedules by time and key in order to keep test results deterministic.
				if !cmp.Equal(want, popped, opts...) {
					t.Errorf("output not equal\ndiff = %v", cmp.Diff(want, popped, opts...))
				}

				// Bump the schedules to add them back into the heap.
				for _, s := range pop.Want {
					if _, err := sched.Bump(s.JobConfig, pop.At); err != nil {
						t.Errorf("cannot bump job config %v at %v: %v", makeKey(s.JobConfig), pop.At, err)
					}
				}
			}
		})
	}
}

func TestSchedule_Sequence(t *testing.T) {
	tests := []struct {
		name      string
		jobConfig *execution.JobConfig
		cases     []time.Time
		fromTime  time.Time
		cfg       *configv1alpha1.CronExecutionConfig
	}{
		{
			name:      "No schedules",
			jobConfig: jobConfigNotScheduled,
			cases: []time.Time{
				{},
			},
		},
		{
			name:      "Schedule every minute",
			jobConfig: jobConfigEveryMinute,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:07:00Z"),
				testutils.Mktime("2021-02-09T04:08:00Z"),
				testutils.Mktime("2021-02-09T04:09:00Z"),
			},
		},
		{
			name:      "No future schedules",
			jobConfig: jobConfigPointInTime,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T12:00:00Z"),
				{}, // should return zero time
			},
		},
		{
			name:      "No future schedules with LastScheduled",
			jobConfig: jobConfigPointInTimeWithAlreadySetLastScheduleTime,
			fromTime:  testutils.Mktime("2021-10-28T06:00:00+08:00"),
			cases: []time.Time{
				{}, // should return zero time
			},
		},
		{
			name:      "Start from LastScheduled",
			jobConfig: jobConfigWithLastScheduleTime,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:03:00Z"),
				testutils.Mktime("2021-02-09T04:04:00Z"),
				testutils.Mktime("2021-02-09T04:05:00Z"),
				testutils.Mktime("2021-02-09T04:06:00Z"),
			},
		},
		{
			name:      "Enforce maximum LastScheduled ago",
			jobConfig: jobConfigWithLastScheduleTimeLongAgo,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:02:00Z"),
				testutils.Mktime("2021-02-09T04:03:00Z"),
				testutils.Mktime("2021-02-09T04:04:00Z"),
				testutils.Mktime("2021-02-09T04:05:00Z"),
				testutils.Mktime("2021-02-09T04:06:00Z"),
			},
		},
		{
			name:      "Using custom timezone",
			jobConfig: jobConfigWithTimezone,
			fromTime:  testutils.Mktime("2021-02-09T04:07:00+08:00"),
			cases: []time.Time{
				testutils.Mktime("2021-02-09T06:00:00+08:00"),
				testutils.Mktime("2021-02-09T08:00:00+08:00"),
				testutils.Mktime("2021-02-09T10:00:00+08:00"),
			},
		},
		{
			name:      "Using custom timezone and lastScheduleTime",
			jobConfig: jobConfigWithTimezoneAndLastScheduleTime,
			fromTime:  testutils.Mktime("2021-06-24T17:19:40+08:00"),
			cases: []time.Time{
				testutils.Mktime("2021-06-24T17:00:00+07:00"),
			},
		},
		{
			name:      "Do not back-schedule with LastUpdated",
			jobConfig: jobConfigWithLastScheduledAndUpdated,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:07:00Z"),
				testutils.Mktime("2021-02-09T04:08:00Z"),
				testutils.Mktime("2021-02-09T04:09:00Z"),
			},
		},
		{
			name:      "Respect NotBefore constraint",
			jobConfig: jobConfigWithNotBeforeConstraint,
			cases: []time.Time{
				testutils.Mktime("2021-02-10T04:02:00Z"),
				testutils.Mktime("2021-02-10T04:03:00Z"),
				testutils.Mktime("2021-02-10T04:04:00Z"),
			},
		},
		{
			name:      "Respect NotBefore constraint with LastScheduled",
			jobConfig: jobConfigWithNotBeforeAndLastScheduled,
			cases: []time.Time{
				testutils.Mktime("2021-02-10T04:02:00Z"),
				testutils.Mktime("2021-02-10T04:03:00Z"),
				testutils.Mktime("2021-02-10T04:04:00Z"),
			},
		},
		{
			name:      "Respect NotBefore constraint with LastUpdated",
			jobConfig: jobConfigWithNotBeforeAndLastUpdated,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:07:00Z"),
				testutils.Mktime("2021-02-09T04:08:00Z"),
				testutils.Mktime("2021-02-09T04:09:00Z"),
			},
		},
		{
			name:      "Respect future NotBefore constraint with LastUpdated",
			jobConfig: jobConfigWithFutureNotBeforeAndLastUpdated,
			cases: []time.Time{
				testutils.Mktime("2021-02-10T04:02:00Z"),
				testutils.Mktime("2021-02-10T04:03:00Z"),
				testutils.Mktime("2021-02-10T04:04:00Z"),
			},
		},
		{
			name:      "Respect NotAfter constraint",
			jobConfig: jobConfigWithNotAfterConstraint,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:07:00Z"),
				testutils.Mktime("2021-02-09T04:08:00Z"),
				{}, // no more
			},
		},
		{
			name:      "Respect NotAfter constraint with LastScheduled",
			jobConfig: jobConfigWithNotAfterWithLastScheduled,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:03:00Z"),
				testutils.Mktime("2021-02-09T04:04:00Z"),
				testutils.Mktime("2021-02-09T04:05:00Z"),
				testutils.Mktime("2021-02-09T04:06:00Z"),
				testutils.Mktime("2021-02-09T04:07:00Z"),
				testutils.Mktime("2021-02-09T04:08:00Z"),
				{}, // no more
			},
		},
		{
			name:      "Respect NotAfter constraint with LastUpdated",
			jobConfig: jobConfigWithNotAfterWithLastUpdated,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:07:00Z"),
				testutils.Mktime("2021-02-09T04:08:00Z"),
				{}, // no more
			},
		},
		{
			name:      "Support multiple cron expressions",
			jobConfig: multiCronJobConfig,
			fromTime:  testutils.Mktime("2021-02-09T18:47:35Z"),
			cases: []time.Time{
				testutils.Mktime("2021-02-09T18:50:00Z"),
				testutils.Mktime("2021-02-09T18:55:00Z"),
				testutils.Mktime("2021-02-09T19:00:00Z"),
				testutils.Mktime("2021-02-09T20:00:00Z"),
				testutils.Mktime("2021-02-09T21:00:00Z"),
			},
		},
		{
			name:      "Support multiple cron expressions #2",
			jobConfig: multiCronJobConfig,
			fromTime:  testutils.Mktime("2021-02-09T07:59:59Z"),
			cases: []time.Time{
				testutils.Mktime("2021-02-09T08:00:00Z"),
				testutils.Mktime("2021-02-09T09:00:00Z"),
				testutils.Mktime("2021-02-09T10:00:00Z"),
				testutils.Mktime("2021-02-09T10:05:00Z"),
				testutils.Mktime("2021-02-09T10:10:00Z"),
			},
		},
		{
			name:      "Do not repeat overlapping cron expressions",
			jobConfig: multiCronJobConfigWithOverlap,
			fromTime:  testutils.Mktime("2021-02-09T18:47:35Z"),
			cases: []time.Time{
				testutils.Mktime("2021-02-09T18:50:00Z"),
				testutils.Mktime("2021-02-09T18:55:00Z"),
				testutils.Mktime("2021-02-09T19:00:00Z"),
				testutils.Mktime("2021-02-09T20:00:00Z"),
				testutils.Mktime("2021-02-09T21:00:00Z"),
			},
		},
		{
			name:      "Do not repeat overlapping cron expressions #2",
			jobConfig: multiCronJobConfigWithOverlap,
			fromTime:  testutils.Mktime("2021-02-09T07:59:59Z"),
			cases: []time.Time{
				testutils.Mktime("2021-02-09T08:00:00Z"),
				testutils.Mktime("2021-02-09T09:00:00Z"),
				testutils.Mktime("2021-02-09T10:00:00Z"),
				testutils.Mktime("2021-02-09T10:05:00Z"),
				testutils.Mktime("2021-02-09T10:10:00Z"),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Change popTime if specified.
			now := testutils.Mktime(now)
			if !tt.fromTime.IsZero() {
				now = tt.fromTime
			}

			// Update popTime timezone if specified.
			if tt.jobConfig.Spec.Schedule != nil && tt.jobConfig.Spec.Schedule.Cron.Timezone != "" {
				tz, err := tzutils.ParseTimezone(tt.jobConfig.Spec.Schedule.Cron.Timezone)
				if err != nil {
					t.Errorf(`failed to parse timezone "%v": %v`, tt.jobConfig.Spec.Schedule.Cron.Timezone, err)
					return
				}
				now = now.In(tz)
			}

			// Use UTC time for the fake clock.
			fakeClock := fakeclock.NewFakeClock(now.In(time.UTC))

			sched, err := cronschedule.New(
				[]*execution.JobConfig{tt.jobConfig},
				cronschedule.WithClock(fakeClock),
				cronschedule.WithConfigLoader(&configLoader{cfg: tt.cfg}),
			)
			if err != nil {
				t.Errorf("cannot initialize schedule: %v", err)
				return
			}

			wantName, err := cache.MetaNamespaceKeyFunc(tt.jobConfig)
			if err != nil {
				t.Errorf("cannot get namespaced name: %v", err)
				return
			}

			for _, wantNext := range tt.cases {
				// NOTE(irvinlim): Use wantNext for popTime, because we want to ensure that it
				// would pop the schedule.
				name, next, ok := sched.Pop(wantNext)

				// Should not have next schedule time.
				if wantNext.IsZero() {
					if ok {
						t.Errorf("wanted zero time, got next time: %v", next)
					}
					continue
				}

				if !ok {
					t.Errorf("Pop() returned false, wanted %v, %v", wantName, wantNext)
				} else if name != wantName || !wantNext.Equal(next) {
					t.Errorf("Pop() = %v %v, wanted %v %v", name, next, wantName, wantNext)
				}

				_, err = sched.Bump(tt.jobConfig, next)
				if err != nil {
					t.Errorf("error bumping next schedule time: %v", err)
					return
				}
			}
		})
	}
}

func TestSchedule_FlushNextScheduleTime(t *testing.T) {
	now := testutils.Mktime(now)
	fakeClock := fakeclock.NewFakeClock(now)
	jobConfig := &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-to-be-updated",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "0 0/5 * * * ? *",
				},
			},
		},
	}

	sched, err := cronschedule.New([]*execution.JobConfig{jobConfig}, cronschedule.WithClock(fakeClock))
	if err != nil {
		t.Fatalf("cannot initialize schedule: %v", err)
	}

	// No next schedule yet
	_, _, ok := sched.Pop(testutils.Mktime("2021-02-09T04:07:00Z"))
	assert.False(t, ok)

	// Pop and bump once time has passed
	key, ts, ok := sched.Pop(testutils.Mktime("2021-02-09T04:10:01Z"))
	assert.True(t, ok)
	assert.True(t, ts.Equal(testutils.Mktime("2021-02-09T04:10:00Z")))
	assert.Equal(t, jobConfig.Name, key)
	_, err = sched.Bump(jobConfig, testutils.Mktime("2021-02-09T04:10:00Z"))
	assert.NoError(t, err)

	// Update job config cron schedule
	newJobConfig := jobConfig.DeepCopy()
	newJobConfig.Spec.Schedule.Cron.Expression = "0 0/1 * * * ? *"

	// No next schedule yet without flushing (it should use 04:15:00)
	_, _, ok = sched.Pop(testutils.Mktime("2021-02-09T04:11:00Z"))
	assert.False(t, ok)

	// Delete the JobConfig from the schedule
	assert.NoError(t, sched.Delete(jobConfig))

	// Add it back
	_, err = sched.Bump(newJobConfig, testutils.Mktime("2021-02-09T04:10:01Z"))
	assert.NoError(t, err)

	// Now it should schedule at the next minute
	key, ts, ok = sched.Pop(testutils.Mktime("2021-02-09T04:11:00Z"))
	assert.True(t, ok)
	assert.True(t, ts.Equal(testutils.Mktime("2021-02-09T04:11:00Z")))
	assert.Equal(t, jobConfig.Name, key)
}

type configLoader struct {
	cfg *configv1alpha1.CronExecutionConfig
}

var _ cronschedule.Config = (*configLoader)(nil)

func (c *configLoader) Cron() (*configv1alpha1.CronExecutionConfig, error) {
	cfg := c.cfg
	if cfg == nil {
		cfg = &configv1alpha1.CronExecutionConfig{}
	}
	return cfg, nil
}

func makeKey(jobConfig *execution.JobConfig) string {
	key, err := cache.MetaNamespaceKeyFunc(jobConfig)
	if err != nil {
		panic(err)
	}
	return key
}

func TestSchedule_Bump(t *testing.T) {
	tests := []struct {
		name      string
		jobConfig *execution.JobConfig
		fromTime  time.Time
		want      time.Time
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:      "bump job config without schedule",
			jobConfig: &execution.JobConfig{},
			fromTime:  testutils.Mktime(now),
		},
		{
			name: "bump job config with disabled schedule",
			jobConfig: &execution.JobConfig{
				Spec: execution.JobConfigSpec{
					Schedule: &execution.ScheduleSpec{
						Cron: &execution.CronSchedule{
							Expression: "0 0/5 * * * ? *",
						},
						Disabled: true,
					},
				},
			},
			fromTime: testutils.Mktime(now),
		},
		{
			name: "bump job config without timezone",
			jobConfig: &execution.JobConfig{
				Spec: execution.JobConfigSpec{
					Schedule: &execution.ScheduleSpec{
						Cron: &execution.CronSchedule{
							Expression: "0 0/5 * * * ? *",
						},
					},
				},
			},
			fromTime: testutils.Mktime(now),
			want:     testutils.Mktime("2021-02-09T04:10:00Z"),
		},
		{
			name: "bump job config with timezone",
			jobConfig: &execution.JobConfig{
				Spec: execution.JobConfigSpec{
					Schedule: &execution.ScheduleSpec{
						Cron: &execution.CronSchedule{
							Expression: "0 0/5 * * * ? *",
							Timezone:   "UTC+08:00",
						},
					},
				},
			},
			fromTime: testutils.Mktime(now),
			want:     testutils.Mktime("2021-02-09T12:10:00+08:00"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := testutils.Mktime(now)
			fakeClock := fakeclock.NewFakeClock(now)
			sched, err := cronschedule.New([]*execution.JobConfig{tt.jobConfig}, cronschedule.WithClock(fakeClock))
			if err != nil {
				t.Fatalf("cannot initialize schedule: %v", err)
			}
			got, err := sched.Bump(tt.jobConfig, tt.fromTime)
			if testutils.WantError(t, tt.wantErr, err, fmt.Sprintf("Bump(%v)", tt.fromTime)) {
				return
			}
			assert.True(t, tt.want.Equal(got), fmt.Sprintf("Bump(%v): want %v, got %v", tt.fromTime, tt.want, got))
		})
	}
}
