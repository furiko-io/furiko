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

package job_test

import (
	"testing"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/execution/job"
)

func TestIsActive(t *testing.T) {
	tests := []struct {
		name string
		rj   *execution.Job
		want bool
	}{
		{
			name: "not started",
			rj:   &execution.Job{},
			want: false,
		},
		{
			name: "not started with status",
			rj: &execution.Job{
				Status: execution.JobStatus{
					Phase: execution.JobStarting,
				},
			},
			want: false,
		},
		{
			name: "no phase",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
				},
			},
			want: true,
		},
		{
			name: "JobStarting",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobStarting,
				},
			},
			want: true,
		},
		{
			name: "JobPending",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobPending,
				},
			},
			want: true,
		},
		{
			name: "JobRunning",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobRunning,
				},
			},
			want: true,
		},
		{
			name: "JobRetryBackoff",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobRetryBackoff,
				},
			},
			want: true,
		},
		{
			name: "JobRetrying",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobRetrying,
				},
			},
			want: true,
		},
		{
			name: "JobKilling",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobKilling,
				},
			},
			want: true,
		},
		{
			name: "JobAdmissionError",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobAdmissionError,
				},
			},
			want: false,
		},
		{
			name: "JobFinishedUnknown",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobFinishedUnknown,
				},
			},
			want: false,
		},
		{
			name: "JobSucceeded",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobSucceeded,
				},
			},
			want: false,
		},
		{
			name: "JobKilled",
			rj: &execution.Job{
				Status: execution.JobStatus{
					StartTime: &startTime,
					Phase:     execution.JobKilled,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := job.IsActive(tt.rj); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}
