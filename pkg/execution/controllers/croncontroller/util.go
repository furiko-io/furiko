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

package croncontroller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/tools/cache"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/cmp"
)

const (
	// defaultTimezone is the default timezone value that will be used if there is
	// no timezone configuration for the JobConfig or a default value set for the
	// controller.
	defaultTimezone = "UTC"
)

// IsScheduleEqual returns true if the ScheduleSpec is not equal and should be updated.
// This equality check is only true in the context of the CronController.
func IsScheduleEqual(orig, updated *execution.ScheduleSpec) (bool, error) {
	// We simply compare all fields here.
	return cmp.IsJSONEqual(orig, updated)
}

// JobConfigKeyFunc returns a key to return the key for a JobConfig with the scheduled timestamp.
func JobConfigKeyFunc(config *execution.JobConfig, scheduleTime time.Time) (string, error) {
	// Get key using default KeyFunc.
	key, err := cache.MetaNamespaceKeyFunc(config)
	if err != nil {
		return "", err
	}

	// Add Unix timestamp (seconds) in key.
	wrappedKey := fmt.Sprintf("%v.%v", key, scheduleTime.Unix())
	return wrappedKey, nil
}

// SplitJobConfigKeyName splits a key into name and scheduled timestamp for a JobConfig.
func SplitJobConfigKeyName(key string) (name string, ts time.Time, err error) {
	// Split out tokens.
	tokens := strings.Split(key, ".")
	if len(tokens) < 2 {
		err = fmt.Errorf("invalid key: %v", key)
		return
	}

	// Parse timestamp
	ts, err = ParseUnix(tokens[len(tokens)-1])
	if err != nil {
		return
	}

	// Get key
	name = strings.Join(tokens[:len(tokens)-1], ".")

	return name, ts, nil
}

// ParseUnix parses a Unix timestamp (in seconds) from a string.
func ParseUnix(unixString string) (time.Time, error) {
	scheduleTimeUnix, err := strconv.Atoi(unixString)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid unix timestamp: %v", unixString)
	}
	return time.Unix(int64(scheduleTimeUnix), 0), nil
}
