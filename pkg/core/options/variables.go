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
	"fmt"
	"regexp"
	"strings"
)

// SubstituteVariables will perform variable substitution according to a substitution map.
// Example: SubstituteVariables("echo ${job.id}", map[string]string{"job.id": "smm-api-test.v2.123"})
func SubstituteVariables(target string, submap map[string]string) string {
	for name, value := range submap {
		search := fmt.Sprintf("${%v}", name)
		target = strings.ReplaceAll(target, search, value)
	}
	return target
}

// SubstituteVariableMaps will perform variable substitution according to a list of substitution maps,
// in order of priority, most important substitution map first.
// Finally, it will substitute empty string for all the given prefixes.
func SubstituteVariableMaps(target string, submaps []map[string]string, prefixes []string) string {
	for _, subs := range submaps {
		target = SubstituteVariables(target, subs)
	}
	return SubstituteEmptyStringForPrefixes(target, prefixes)
}

// SubstituteEmptyStringForPrefixes will perform substitution of any variables beginning with any of the given prefixes
// with an empty string.
// This should be performed as the last step of the substitution pipeline, since it would completely remove any
// variables that are not yet substituted.
// Example: SubstituteEmptyStringForPrefixes("echo ${job.unknown_variable};", []string{"job."}) => "echo ;"
func SubstituteEmptyStringForPrefixes(target string, prefixes []string) string {
	for _, prefix := range prefixes {
		// Drop trailing dot if specified, we will add it later.
		prefix = strings.TrimSuffix(prefix, ".")
		regex := regexp.MustCompile(fmt.Sprintf(`\$\{%v\.[^}]+\}`, prefix))
		target = regex.ReplaceAllString(target, "")
	}
	return target
}

// MergeSubstitutions merges the substitution maps, with lowest priority first.
// Does not modify the maps in-place.
func MergeSubstitutions(paramMaps ...map[string]string) map[string]string {
	newParams := make(map[string]string)
	for _, params := range paramMaps {
		for k, v := range params {
			newParams[k] = v
		}
	}
	return newParams
}
