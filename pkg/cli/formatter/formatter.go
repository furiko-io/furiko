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

package formatter

import (
	"strconv"
)

// FormatBool formats a bool as yes/no.
func FormatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// FormatInt formats an int.
func FormatInt(i int) string {
	return strconv.Itoa(i)
}

// FormatInt64 formats an int64.
func FormatInt64(i int64) string {
	return strconv.Itoa(int(i))
}
