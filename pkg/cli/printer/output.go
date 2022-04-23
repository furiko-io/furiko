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

package printer

// OutputFormat specifies a format to print output.
type OutputFormat string

const (
	OutputFormatPretty OutputFormat = "pretty"
	OutputFormatName   OutputFormat = "name"
	OutputFormatJSON   OutputFormat = "json"
	OutputFormatYAML   OutputFormat = "yaml"
)

// AllOutputFormats is a list of all possible output formats.
var AllOutputFormats = []OutputFormat{
	OutputFormatPretty,
	OutputFormatName,
	OutputFormatJSON,
	OutputFormatYAML,
}

func (f OutputFormat) IsValid() bool {
	for _, format := range AllOutputFormats {
		if format == f {
			return true
		}
	}
	return false
}

// GetAllOutputFormatStrings returns a list of all possible output formats as a
// slice of strings.
func GetAllOutputFormatStrings() []string {
	strs := make([]string, 0, len(AllOutputFormats))
	for _, format := range AllOutputFormats {
		strs = append(strs, string(format))
	}
	return strs
}
