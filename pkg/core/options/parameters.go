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

package options

import (
	"strings"
)

// FilterParametersWithPrefix filters in keys from the given parameter map which have the provided prefix.
func FilterParametersWithPrefix(parameters map[string]string, prefix string) map[string]string {
	return FilterParametersWithPrefixes(parameters, prefix)
}

// FilterParametersWithPrefixes filters in keys from the given parameter map which have any of the provided prefixes.
func FilterParametersWithPrefixes(parameters map[string]string, prefixes ...string) map[string]string {
	filtered := make(map[string]string)
	for k, v := range parameters {
		if HasAnyPrefix(k, prefixes) {
			filtered[k] = v
		}
	}
	return filtered
}

// HasAnyPrefix tests whether the string s begins with at least one prefix in prefixes.
func HasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
