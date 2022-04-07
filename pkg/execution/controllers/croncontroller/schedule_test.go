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

package croncontroller_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/tzutils"
	"github.com/furiko-io/furiko/pkg/execution/controllers/croncontroller"
	"github.com/furiko-io/furiko/pkg/execution/util/cronparser"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	now = "2021-02-09T04:06:13.234Z"
)

var (
	lastScheduleTime     = metav1.NewTime(testutils.Mktime("2021-02-09T04:02:00Z"))
	longLastScheduleTime = metav1.NewTime(testutils.Mktime("2021-02-09T02:46:00Z"))

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
			LastScheduleTime: testutils.Mkmtimep("2021-10-27T22:15:00Z"),
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
			LastScheduleTime: &lastScheduleTime,
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
			LastScheduleTime: &longLastScheduleTime,
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
			LastScheduleTime: testutils.Mkmtimep("2021-06-24T09:00:00Z"),
		},
	}

	jobConfigWithLastScheduleTimeAndNotBefore = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-with-last-schedule-time-and-not-before",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotBefore: testutils.Mkmtimep(now),
				},
			},
		},
		Status: execution.JobConfigStatus{
			LastScheduleTime: &longLastScheduleTime,
		},
	}
)

func TestSchedule(t *testing.T) {
	tests := []struct {
		name      string
		jobConfig *execution.JobConfig
		cases     []time.Time
		fromTime  time.Time
	}{
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
			name:      "No future schedules with LastScheduleTime",
			jobConfig: jobConfigPointInTimeWithAlreadySetLastScheduleTime,
			fromTime:  testutils.Mktime("2021-10-28T06:00:00+08:00"),
			cases: []time.Time{
				{}, // should return zero time
			},
		},
		{
			name:      "Start from LastScheduleTime",
			jobConfig: jobConfigWithLastScheduleTime,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:03:00Z"),
				testutils.Mktime("2021-02-09T04:04:00Z"),
				testutils.Mktime("2021-02-09T04:05:00Z"),
				testutils.Mktime("2021-02-09T04:06:00Z"),
			},
		},
		{
			name:      "Enforce maximum LastScheduleTime ago",
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
			name:      "Respect ScheduleNotBeforeTimestamp",
			jobConfig: jobConfigWithLastScheduleTimeAndNotBefore,
			cases: []time.Time{
				testutils.Mktime("2021-02-09T04:07:00Z"),
				testutils.Mktime("2021-02-09T04:08:00Z"),
				testutils.Mktime("2021-02-09T04:09:00Z"),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Change fromTime if specified.
			now := testutils.Mktime(now)
			if !tt.fromTime.IsZero() {
				now = tt.fromTime
			}

			// Update fromTime timezone if specified.
			if tt.jobConfig.Spec.Schedule.Cron.Timezone != "" {
				tz, err := tzutils.ParseTimezone(tt.jobConfig.Spec.Schedule.Cron.Timezone)
				if err != nil {
					t.Errorf(`failed to parse timezone "%v": %v`, tt.jobConfig.Spec.Schedule.Cron.Timezone, err)
					return
				}
				now = now.In(tz)
			}

			// Reset the timezone to the local timezone to avoid false positives.
			croncontroller.Clock = clock.NewFakeClock(now.In(time.Local))

			ctrlContext := mock.NewContext()
			if err := ctrlContext.Start(context.Background()); err != nil {
				t.Fatalf("cannot start context: %v", err)
			}

			cfg, err := ctrlContext.Configs().CronController()
			if err != nil {
				t.Fatalf("cannot get controller configuration: %v", err)
			}
			schedule := croncontroller.NewSchedule(ctrlContext)
			parser := cronparser.NewParser(cfg)
			expr, err := parser.Parse(tt.jobConfig.Spec.Schedule.Cron.Expression, "")
			if err != nil {
				t.Errorf("cannot parse cron %v: %v", tt.jobConfig.Spec.Schedule.Cron.Expression, err)
				return
			}

			for _, want := range tt.cases {
				next := schedule.GetNextScheduleTime(tt.jobConfig, now, expr)
				if !next.Equal(want) {
					t.Errorf("expected next to be %v, got %v", want, next)
				}
				schedule.BumpNextScheduleTime(tt.jobConfig, next, expr)
			}
		})
	}
}

func TestSchedule_FlushNextScheduleTime(t *testing.T) {
	now := testutils.Mktime(now)
	croncontroller.Clock = clock.NewFakeClock(now)
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

	ctrlContext := mock.NewContext()
	if err := ctrlContext.Start(context.Background()); err != nil {
		t.Fatalf("cannot start context: %v", err)
	}

	cfg, err := ctrlContext.Configs().CronController()
	if err != nil {
		t.Fatalf("cannot get controller configuration: %v", err)
	}
	schedule := croncontroller.NewSchedule(ctrlContext)
	parser := cronparser.NewParser(cfg)

	expr, err := parser.Parse(jobConfig.Spec.Schedule.Cron.Expression, "")
	if err != nil {
		t.Fatalf("cannot parse cron %v: %v", jobConfig.Spec.Schedule.Cron.Expression, err)
	}

	// Bump next schedule time
	next := schedule.GetNextScheduleTime(jobConfig, now, expr)
	want := testutils.Mktime("2021-02-09T04:10:00Z")
	if !next.Equal(want) {
		t.Errorf("expected next to be %v, got %v", want, next)
		return
	}
	schedule.BumpNextScheduleTime(jobConfig, next, expr)

	// Update job config cron schedule
	jobConfig.Spec.Schedule.Cron.Expression = "0 0/1 * * * ? *"
	expr, err = parser.Parse(jobConfig.Spec.Schedule.Cron.Expression, "")
	if err != nil {
		t.Fatalf("cannot parse cron %v: %v", jobConfig.Spec.Schedule.Cron.Expression, err)
	}

	// Should still use old schedule because it's cached
	next2 := schedule.GetNextScheduleTime(jobConfig, next, expr)
	want2 := testutils.Mktime("2021-02-09T04:15:00Z")
	if !next2.Equal(want2) {
		t.Errorf("expected next to be %v, got %v", want2, next2)
		return
	}

	// Flush the time, should use new schedule now
	flushAndCheck := func(want time.Time) error {
		// Flush
		schedule.FlushNextScheduleTime(jobConfig)

		// Check next
		next = schedule.GetNextScheduleTime(jobConfig, next, expr)
		if !next.Equal(want) {
			return fmt.Errorf("expected next to be %v, got %v", want, next)
		}
		schedule.BumpNextScheduleTime(jobConfig, next, expr)
		return nil
	}
	if err := flushAndCheck(testutils.Mktime("2021-02-09T04:11:00Z")); err != nil {
		t.Error(err)
		return
	}
	if err := flushAndCheck(testutils.Mktime("2021-02-09T04:12:00Z")); err != nil { // idempotent
		t.Error(err)
		return
	}
}
