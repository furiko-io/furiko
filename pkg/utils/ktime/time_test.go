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

package ktime_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

var (
	ts1 = metav1.Unix(1626244260, 0)
	ts2 = metav1.Unix(1626244500, 0)
)

func TestTimeMax(t *testing.T) {
	type args struct {
		ts1 *metav1.Time
		ts2 *metav1.Time
	}
	tests := []struct {
		name string
		args args
		want *metav1.Time
	}{
		{
			name: "nil timestamps",
			args: args{
				ts1: nil,
				ts2: nil,
			},
			want: nil,
		},
		{
			name: "nil t1",
			args: args{
				ts1: nil,
				ts2: &ts2,
			},
			want: &ts2,
		},
		{
			name: "nil t2",
			args: args{
				ts1: &ts1,
				ts2: nil,
			},
			want: &ts1,
		},
		{
			name: "greater time",
			args: args{
				ts1: &ts1,
				ts2: &ts2,
			},
			want: &ts2,
		},
		{
			name: "switch around",
			args: args{
				ts1: &ts2,
				ts2: &ts1,
			},
			want: &ts2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ktime.TimeMax(tt.args.ts1, tt.args.ts2); !tt.want.Equal(got) {
				t.Errorf("TimeMax() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnixZero(t *testing.T) {
	tests := []struct {
		name string
		ts   metav1.Time
		want bool
	}{
		{
			name: "nonzero time",
			ts:   ts1,
			want: false,
		},
		{
			name: "zero time",
			want: true,
		},
		{
			name: "unix zero",
			ts:   metav1.Unix(0, 0),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ktime.IsUnixZero(&tt.ts); got != tt.want {
				t.Errorf("IsUnixZero() = %v, want %v", got, tt.want)
			}
		})
	}
}
