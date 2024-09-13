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
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/croncontroller"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	scheduleUnix  = 1604188800
	scheduleTime  = "2020-11-01T00:00:00Z"
	testNamespace = "test"
	testName      = "sample-job.1605774500690"
)

func TestJobConfigKeyFunc(t *testing.T) {
	tests := []struct {
		name         string
		scheduleTime time.Time
		want         string
		wantErr      bool
	}{
		{
			name:         "basic test",
			scheduleTime: testutils.Mktime(scheduleTime),
			want:         fmt.Sprintf("%v/%v.%v", testNamespace, testName, scheduleUnix),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobConfig := &execution.JobConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testName,
				},
			}
			got, err := croncontroller.JobConfigKeyFunc(jobConfig, tt.scheduleTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("JobConfigKeyFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("JobConfigKeyFunc() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitJobConfigKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		wantName string
		wantTS   time.Time
		wantErr  bool
	}{
		{
			name:     "basic test",
			key:      fmt.Sprintf("%v.%v", testName, scheduleUnix),
			wantName: testName,
			wantTS:   testutils.Mktime(scheduleTime),
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "invalid key",
			key:     "alskdjl",
			wantErr: true,
		},
		{
			name:    "invalid timestamp",
			key:     fmt.Sprintf("%v.invalid", testName),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotTS, err := croncontroller.SplitJobConfigKeyName(tt.key)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("SplitJobConfigKeyName() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if tt.wantErr {
				t.Errorf("SplitJobConfigKeyName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotName != tt.wantName {
				t.Errorf("SplitJobConfigKeyName() gotName = %v, want %v", gotName, tt.wantName)
			}
			if !gotTS.Equal(tt.wantTS) {
				t.Errorf("SplitJobConfigKeyName() gotTS = %v, want %v", gotTS, tt.wantTS)
			}
		})
	}
}
