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
)

func TestListTimezones(t *testing.T) {
	zones, err := ListTimezones()
	if err != nil {
		t.Fatalf("cannot list timezones: %v", err)
	}

	zoneMap := make(map[string]struct{})
	for _, zone := range zones {
		zoneMap[zone] = struct{}{}
	}

	// Try parsing all of them
	for _, tz := range zones {
		_, err := ParseTimezone(tz)
		if err != nil {
			t.Errorf("cannot parse %v: %v", tz, err)
		}
	}

	// Assert that some well-known ones exist.
	wellKnown := []string{
		"Africa/Cairo",
		"America/New_York",
		"Asia/Singapore",
		"Europe/London",
	}
	for _, zone := range wellKnown {
		if _, ok := zoneMap[zone]; !ok {
			t.Errorf("expected %v to exist in timezone list", zone)
		}
	}
}
