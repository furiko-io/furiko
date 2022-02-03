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
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// MakeDefaultOptions returns a substitution map containing default values for all options
// in the given OptionSpec.
func MakeDefaultOptions(cfg *execution.OptionSpec) (map[string]string, error) {
	opts := make(map[string]string)
	if cfg == nil {
		return opts, nil
	}

	for _, option := range cfg.Options {
		// Evaluate default option variables.
		val, err := EvaluateOptionDefault(option)
		if err != nil {
			return nil, err
		}
		opts[MakeOptionVariableName(option)] = val
	}

	return opts, nil
}
