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

package jobconfig_test

import (
	"testing"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
)

var (
	cronSchedule1 = execution.CronSchedule{
		Expression: "0 5 * * *",
	}
)

func TestGetState(t *testing.T) {
	tests := []struct {
		name string
		rjc  *execution.JobConfig
		want execution.JobConfigState
	}{
		{
			name: "Ready",
			rjc:  &execution.JobConfig{},
			want: execution.JobConfigReady,
		},
		{
			name: "Ready, disabled without cronSchedule",
			rjc: &execution.JobConfig{
				Spec: execution.JobConfigSpec{
					Schedule: &execution.ScheduleSpec{
						Disabled: true,
					},
				},
			},
			want: execution.JobConfigReady,
		},
		{
			name: "ReadyEnabled",
			rjc: &execution.JobConfig{
				Spec: execution.JobConfigSpec{
					Schedule: &execution.ScheduleSpec{
						Cron: &cronSchedule1,
					},
				},
			},
			want: execution.JobConfigReadyEnabled,
		},
		{
			name: "ReadyDisabled",
			rjc: &execution.JobConfig{
				Spec: execution.JobConfigSpec{
					Schedule: &execution.ScheduleSpec{
						Cron:     &cronSchedule1,
						Disabled: true,
					},
				},
			},
			want: execution.JobConfigReadyDisabled,
		},
		{
			name: "Executing",
			rjc: &execution.JobConfig{
				Spec: execution.JobConfigSpec{
					Schedule: &execution.ScheduleSpec{
						Cron:     &cronSchedule1,
						Disabled: true,
					},
				},
				Status: execution.JobConfigStatus{
					Queued: 1,
					Active: 1,
				},
			},
			want: execution.JobConfigExecuting,
		},
		{
			name: "JobQueued",
			rjc: &execution.JobConfig{
				Spec: execution.JobConfigSpec{
					Schedule: &execution.ScheduleSpec{
						Cron: &cronSchedule1,
					},
				},
				Status: execution.JobConfigStatus{
					Queued: 1,
				},
			},
			want: execution.JobConfigJobQueued,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := jobconfig.GetState(tt.rjc); got != tt.want {
				t.Errorf("GetState() = %v, want %v", got, tt.want)
			}
		})
	}
}
