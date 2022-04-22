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

// MutateDefaultingOptionSpec populates default fields for an OptionSpec in-place.
func MutateDefaultingOptionSpec(spec *execution.OptionSpec) *execution.OptionSpec {
	newSpec := spec.DeepCopy()
	for i, option := range spec.Options {
		newSpec.Options[i] = GetDefaultingOption(option)
	}
	return newSpec
}

// GetDefaultingOption returns an option with default fields populated.
func GetDefaultingOption(option execution.Option) execution.Option {
	switch option.Type {
	case execution.OptionTypeBool:
		return GetDefaultingOptionBool(option)
	}
	return option
}

// GetDefaultingOptionBool returns a Bool option with default fields populated.
func GetDefaultingOptionBool(option execution.Option) execution.Option {
	newOption := option.DeepCopy()
	if newOption.Bool == nil {
		newOption.Bool = &execution.BoolOptionConfig{}
	}
	if newOption.Bool.Format == "" {
		newOption.Bool.Format = execution.BoolOptionFormatTrueFalse
	}
	return *newOption
}
