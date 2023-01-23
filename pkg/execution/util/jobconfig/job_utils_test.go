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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
)

var (
	timestring1 = "1626244260"
	timestring2 = "1626244380"
	timestring3 = "1626244500"
	metatime1   = metav1.Unix(1626244260, 0)
	metatime2   = metav1.Unix(1626244380, 0)
	metatime3   = metav1.Unix(1626244500, 0)
)

func TestGetLastScheduleTime(t *testing.T) {
	tests := []struct {
		name string
		jobs []*execution.Job
		want *metav1.Time
	}{
		{
			name: "empty list",
			jobs: nil,
			want: nil,
		},
		{
			name: "no annotation",
			jobs: []*execution.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job1",
					},
				},
			},
			want: nil,
		},
		{
			name: "single item",
			jobs: []*execution.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job1",
						Annotations: map[string]string{
							jobconfig.AnnotationKeyScheduleTime: timestring1,
						},
					},
				},
			},
			want: &metatime1,
		},
		{
			name: "multiple items",
			jobs: []*execution.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job1",
						Annotations: map[string]string{
							jobconfig.AnnotationKeyScheduleTime: timestring1,
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job2",
						Annotations: map[string]string{
							jobconfig.AnnotationKeyScheduleTime: timestring2,
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job3",
						Annotations: map[string]string{
							jobconfig.AnnotationKeyScheduleTime: timestring3,
						},
					},
				},
			},
			want: &metatime3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jobconfig.GetLastScheduleTime(tt.jobs); !tt.want.Equal(got) {
				t.Errorf("GetLastScheduleTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLastStartTime(t *testing.T) {
	tests := []struct {
		name string
		jobs []*execution.Job
		want *metav1.Time
	}{
		{
			name: "empty list",
			jobs: nil,
			want: nil,
		},
		{
			name: "no status",
			jobs: []*execution.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job1",
					},
				},
			},
			want: nil,
		},
		{
			name: "job is started",
			jobs: []*execution.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job1",
					},
					Status: execution.JobStatus{
						StartTime: &metatime1,
					},
				},
			},
			want: &metatime1,
		},
		{
			name: "don't take non-started job",
			jobs: []*execution.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job1",
					},
					Status: execution.JobStatus{
						StartTime: &metatime1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job2",
					},
				},
			},
			want: &metatime1,
		},
		{
			name: "multiple items",
			jobs: []*execution.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job1",
					},
					Status: execution.JobStatus{
						StartTime: &metatime1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "job2",
					},
					Status: execution.JobStatus{
						StartTime: &metatime2,
					},
				},
			},
			want: &metatime2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jobconfig.GetLastStartTime(tt.jobs); !tt.want.Equal(got) {
				t.Errorf("GetLastStartTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
