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
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// defaultTimezone is used as a fallback when no timezone is specified.
	defaultTimezone = time.UTC

	regexpNonLeadingTZ      = regexp.MustCompile(`^[+-]\d$`)
	regexpNonLeadingColonTZ = regexp.MustCompile(`^[+-]\d:\d{2}$`)
)

// ParseTimezone attempts to parse a timezone string that may not be tzdata-compliant.
//
// Returns a *time.Location whose name is normalized to either of the following:
// 1. tzdata name, which may be canonical, aliased or deprecated (no normalization)
// 2. Specific UTC offset in the format "UTC-07:00".
//
// This means that ParseTimezone should be more lenient that the list of canonical location names in tzdata,
// and the list of tzdata names (e.g. from zone.tab) is not an exhaustive list of valid inputs to this function.
//
// Note that not all timezone abbreviations can be parsed accurately; many of them are often ambiguous.
// (e.g. CST could refer to any of China Standard Time, Cuban Standard Time or Central Standard Time).
// See https://en.wikipedia.org/wiki/List_of_time_zone_abbreviations for more information.
func ParseTimezone(val string) (*time.Location, error) {
	// If empty, uses a default timezone.
	if val == "" || val == "Local" {
		return defaultTimezone, nil
	}

	// Convert GMT -> UTC
	if val == "GMT" {
		val = "UTC"
	}

	// Try parse tzdata name.
	if loc, err := time.LoadLocation(val); err == nil {
		return loc, nil
	}

	// Try parse UTC/GMT offsets
	if strings.HasPrefix(val, "UTC") || strings.HasPrefix(val, "GMT") {
		offset := val[3:]

		// Support formats without leading zeros.
		if regexpNonLeadingTZ.MatchString(offset) {
			offset = offset[0:1] + "0" + offset[1:]
		} else if regexpNonLeadingColonTZ.MatchString(offset) {
			offset = offset[0:1] + "0" + offset[1:]
		}

		// Use alternative time format parsing.
		if t, err := time.Parse("-07", offset); err == nil {
			return getFixedZoneForUTCOffset(t), nil
		}
		if t, err := time.Parse("-07:00", offset); err == nil {
			return getFixedZoneForUTCOffset(t), nil
		}
		if t, err := time.Parse("-0700", offset); err == nil {
			return getFixedZoneForUTCOffset(t), nil
		}
	}

	return nil, fmt.Errorf("cannot parse \"%v\" as timezone", val)
}

func getFixedZoneForUTCOffset(t time.Time) *time.Location {
	name := fmt.Sprintf("UTC%v", t.Format("-07:00"))
	_, offset := t.Zone()
	return time.FixedZone(name, offset)
}
