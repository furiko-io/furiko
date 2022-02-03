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

package tzutils

import (
	"testing"
	"time"

	"github.com/furiko-io/cronexpr"
)

func TestParseTimezone(t *testing.T) { // nolint: gocognit
	fromTime, err := time.Parse(time.RFC3339, "2021-03-09T15:04:05+08:00")
	if err != nil {
		t.Fatal(err)
	}

	// Additional fromTime for DST test
	fromDSTTime, err := time.Parse(time.RFC3339, "2021-04-09T15:04:05+08:00")
	if err != nil {
		t.Fatal(err)
	}

	cron := cronexpr.MustParse("0 3/6 * * *")

	tests := []struct {
		name         string
		val          string
		wantErr      bool
		wantLocation string
		wantNext     string
		wantDSTNext  string
	}{
		// Default timezones
		{
			name:         "default timezone for empty",
			val:          "",
			wantLocation: defaultTimezone.String(),
			wantNext:     "2021-03-09T09:00:00Z",
		},
		{
			name:         "local timezone",
			val:          "Local",
			wantLocation: defaultTimezone.String(),
			wantNext:     "2021-03-09T09:00:00Z",
		},
		// tzdata names
		{
			name:         "Asia/Jakarta",
			val:          "Asia/Jakarta",
			wantLocation: "Asia/Jakarta",
			wantNext:     "2021-03-09T15:00:00+07:00",
		},
		{
			name:         "America/New_York",
			val:          "America/New_York",
			wantLocation: "America/New_York",
			wantNext:     "2021-03-09T03:00:00-05:00",
			wantDSTNext:  "2021-04-09T09:00:00-04:00",
		},
		// GMT/UTC offsets
		{
			name:         "UTC",
			val:          "UTC",
			wantLocation: "UTC",
			wantNext:     "2021-03-09T09:00:00Z",
		},
		{
			name:         "UTC+1",
			val:          "UTC+1",
			wantLocation: "UTC+01:00",
			wantNext:     "2021-03-09T09:00:00+01:00",
		},
		{
			name:         "UTC+1:00",
			val:          "UTC+1:00",
			wantLocation: "UTC+01:00",
			wantNext:     "2021-03-09T09:00:00+01:00",
		},
		{
			name:         "UTC+01",
			val:          "UTC+01",
			wantLocation: "UTC+01:00",
			wantNext:     "2021-03-09T09:00:00+01:00",
		},
		{
			name:         "UTC+01:00",
			val:          "UTC+01:00",
			wantLocation: "UTC+01:00",
			wantNext:     "2021-03-09T09:00:00+01:00",
		},
		{
			name:         "UTC+0100",
			val:          "UTC+0100",
			wantLocation: "UTC+01:00",
			wantNext:     "2021-03-09T09:00:00+01:00",
		},
		{
			name:         "GMT",
			val:          "GMT",
			wantLocation: "UTC",
			wantNext:     "2021-03-09T09:00:00Z",
		},
		{
			name:         "GMT+01:00",
			val:          "GMT+01:00",
			wantLocation: "UTC+01:00",
			wantNext:     "2021-03-09T09:00:00+01:00",
		},
		{
			name:         "GMT+0100",
			val:          "GMT+0100",
			wantLocation: "UTC+01:00",
			wantNext:     "2021-03-09T09:00:00+01:00",
		},
		// Etc area
		{
			name:         "Etc/GMT",
			val:          "Etc/GMT",
			wantLocation: "Etc/GMT",
			wantNext:     "2021-03-09T09:00:00Z",
		},
		{
			name:         "Etc/GMT-8",
			val:          "Etc/GMT-8",
			wantLocation: "Etc/GMT-8",
			wantNext:     "2021-03-09T21:00:00+08:00",
		},
		// Deprecated country-only aliases
		{
			name:         "Singapore",
			val:          "Singapore",
			wantLocation: "Singapore",
			wantNext:     "2021-03-09T21:00:00+08:00",
		},
		{
			name:         "Japan",
			val:          "Japan",
			wantLocation: "Japan",
			wantNext:     "2021-03-09T21:00:00+09:00",
		},
		// Deprecated US timezone aliases
		{
			name:         "MST",
			val:          "MST",
			wantLocation: "MST",
			wantNext:     "2021-03-09T03:00:00-07:00",
		},
		{
			name:         "MST7MDT",
			val:          "MST7MDT",
			wantLocation: "MST7MDT",
			wantNext:     "2021-03-09T03:00:00-07:00",
			wantDSTNext:  "2021-04-09T03:00:00-06:00",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			loc, err := ParseTimezone(tt.val)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLocation(\"%v\") wantErr = %v, got %v", tt.val, tt.wantErr, err)
			}
			if err != nil {
				return
			}

			if loc.String() != tt.wantLocation {
				t.Errorf("ParseLocation(\"%v\") wantLocation = %v, got %v", tt.val, tt.wantLocation, loc.String())
			}

			if next := cron.Next(fromTime.In(loc)); !next.IsZero() {
				nextFormatted := next.Format(time.RFC3339)
				if nextFormatted != tt.wantNext {
					t.Errorf("ParseLocation(\"%v\") wantNext = %v, got %v", tt.val, tt.wantNext, nextFormatted)
				}
			}

			if tt.wantDSTNext != "" {
				if next := cron.Next(fromDSTTime.In(loc)); !next.IsZero() {
					nextFormatted := next.Format(time.RFC3339)
					if nextFormatted != tt.wantDSTNext {
						t.Errorf("ParseLocation(\"%v\") wantDSTNext = %v, got %v", tt.val, tt.wantDSTNext, nextFormatted)
					}
				}
			}
		})
	}
}
