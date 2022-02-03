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
	"bufio"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
)

var (
	getZoneDirentsOnce sync.Once
	cachedZoneDirents  []string
	getZoneDirentsErr  error
)

// ListTimezones returns a list of timezones known to the system.
func ListTimezones() ([]string, error) {
	getZoneDirentsOnce.Do(func() {
		cachedZoneDirents, getZoneDirentsErr = listTimezonesFromZoneTab()
	})
	return cachedZoneDirents, getZoneDirentsErr
}

// listTimezonesFromZoneTab approximates the behaviour of `timedatectl list-timezones`.
// Parses data from zone.tab.
func listTimezonesFromZoneTab() ([]string, error) {
	var firstErr error
	for _, zoneSource := range zoneSources {
		zonetabPath := path.Join(zoneSource, "zone.tab")

		file, err := os.Open(zonetabPath)
		if err != nil {
			if firstErr == nil && !os.IsNotExist(err) {
				firstErr = err
			}
			continue
		}
		scanner := bufio.NewScanner(file)

		var timezones []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) == 0 || line[0] == '#' {
				continue
			}
			// Format is: country-code coordinates TZ comments
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			timezones = append(timezones, fields[2])
		}

		sort.Strings(timezones)
		return timezones, nil
	}

	if firstErr != nil {
		return nil, firstErr
	}

	return nil, fmt.Errorf("no known zone sources")
}
