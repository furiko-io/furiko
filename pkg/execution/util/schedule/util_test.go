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

package schedule_test

import (
	"testing"
	"time"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/schedule"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	timeNow    = "2021-02-09T04:06:13Z"
	timePast   = "2020-05-10T03:14:15Z"
	timeFuture = "2022-03-14T15:26:35Z"
	notBefore  = "2021-01-01T00:00:00Z"
	notAfter   = "2021-12-31T23:59:59Z"
)

func TestIsAllowed(t *testing.T) {
	tests := []struct {
		name     string
		schedule *execution.ScheduleSpec
		t        time.Time
		want     bool
	}{
		{
			name: "normal schedule",
			schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
			},
			t:    testutils.Mktime(timeNow),
			want: true,
		},
		{
			name: "disabled schedule",
			schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Disabled: true,
			},
			t:    testutils.Mktime(timeNow),
			want: false,
		},
		{
			name: "empty constraints",
			schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{},
			},
			t:    testutils.Mktime(timeNow),
			want: true,
		},
		{
			name: "cannot schedule before notBefore",
			schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotBefore: testutils.Mkmtimep(notBefore),
				},
			},
			t:    testutils.Mktime(timePast),
			want: false,
		},
		{
			name: "can schedule after notBefore",
			schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotBefore: testutils.Mkmtimep(notBefore),
				},
			},
			t:    testutils.Mktime(timeNow),
			want: true,
		},
		{
			name: "cannot schedule after notAfter",
			schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotAfter: testutils.Mkmtimep(notAfter),
				},
			},
			t:    testutils.Mktime(timeFuture),
			want: false,
		},
		{
			name: "can schedule before notAfter",
			schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
				Constraints: &execution.ScheduleContraints{
					NotAfter: testutils.Mkmtimep(notAfter),
				},
			},
			t:    testutils.Mktime(timeNow),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := schedule.IsAllowed(tt.schedule, tt.t); got != tt.want {
				t.Errorf("IsAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
