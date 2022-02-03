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

package leaderelection_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/furiko-io/furiko/pkg/runtime/leaderelection"
)

func TestConfig_Prepare(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *leaderelection.Config
		expected *leaderelection.Config
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: leaderelection.DefaultConfig,
		},
		{
			name:     "zero values",
			cfg:      &leaderelection.Config{},
			expected: leaderelection.DefaultConfig,
		},
		{
			name: "negative values",
			cfg: &leaderelection.Config{
				LeaseDuration: -1,
				RenewDeadline: -2,
				RetryPeriod:   -3,
			},
			expected: leaderelection.DefaultConfig,
		},
		{
			name: "only override zero values",
			cfg: &leaderelection.Config{
				LeaseDuration: time.Minute,
			},
			expected: &leaderelection.Config{
				LeaseNamespace: leaderelection.DefaultConfig.LeaseNamespace,
				LeaseDuration:  time.Minute,
				RenewDeadline:  leaderelection.DefaultConfig.RenewDeadline,
				RetryPeriod:    leaderelection.DefaultConfig.RetryPeriod,
			},
		},
		{
			name: "only override negative values",
			cfg: &leaderelection.Config{
				LeaseDuration: time.Minute,
				RenewDeadline: -2,
				RetryPeriod:   -3,
			},
			expected: &leaderelection.Config{
				LeaseNamespace: leaderelection.DefaultConfig.LeaseNamespace,
				LeaseDuration:  time.Minute,
				RenewDeadline:  leaderelection.DefaultConfig.RenewDeadline,
				RetryPeriod:    leaderelection.DefaultConfig.RetryPeriod,
			},
		},
		{
			name: "do not override any value",
			cfg: &leaderelection.Config{
				LeaseDuration: time.Minute,
				RenewDeadline: time.Second * 45,
				RetryPeriod:   time.Second * 30,
			},
			expected: &leaderelection.Config{
				LeaseNamespace: leaderelection.DefaultConfig.LeaseNamespace,
				LeaseDuration:  time.Minute,
				RenewDeadline:  time.Second * 45,
				RetryPeriod:    time.Second * 30,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg.PrepareValues()
			if !cmp.Equal(tt.expected, cfg) {
				t.Errorf("not equal after Prepare():\ndiff = %v", cmp.Diff(tt.expected, cfg))
			}
		})
	}
}
