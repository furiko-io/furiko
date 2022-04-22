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
	"reflect"

	"k8s.io/apimachinery/pkg/util/validation/field"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	stringsutils "github.com/furiko-io/furiko/pkg/utils/strings"
)

// ValidateOptionSpec validates the OptionSpec and returns a list of errors.
func ValidateOptionSpec(spec *execution.OptionSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec != nil {
		optionNames := make(map[string]struct{}, len(spec.Options))
		for i, option := range spec.Options {
			fldPath := fldPath.Child("options").Index(i)
			if _, ok := optionNames[option.Name]; ok {
				detail := fmt.Sprintf(`duplicate option name: "%v"`, option.Name)
				allErrs = append(allErrs, field.Forbidden(fldPath, detail))
				continue
			}
			optionNames[option.Name] = struct{}{}
			allErrs = append(allErrs, ValidateOption(option, fldPath)...)
		}
	}
	return allErrs
}

// ValidateOption validates the option and returns a list of errors.
func ValidateOption(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateOptionType(option.Type, fldPath)...)
	allErrs = append(allErrs, ValidateOptionName(option.Name, fldPath)...)
	switch option.Type {
	case execution.OptionTypeBool:
		allErrs = append(allErrs, ValidateOptionBool(option, fldPath)...)
	case execution.OptionTypeString:
		allErrs = append(allErrs, ValidateOptionString(option, fldPath)...)
	case execution.OptionTypeSelect:
		allErrs = append(allErrs, ValidateOptionSelect(option, fldPath)...)
	case execution.OptionTypeMulti:
		allErrs = append(allErrs, ValidateOptionMulti(option, fldPath)...)
	case execution.OptionTypeDate:
		allErrs = append(allErrs, ValidateOptionDate(option, fldPath)...)
	}
	return allErrs
}

// ValidateOptionType validates the OptionType and returns a list of errors.
func ValidateOptionType(optionType execution.OptionType, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if optionType == "" {
		allErrs = append(allErrs, field.Required(fldPath, ""))
	}
	if !optionType.IsValid() {
		supportedValues := make([]string, 0, len(execution.OptionTypesAll))
		for _, o := range execution.OptionTypesAll {
			supportedValues = append(supportedValues, string(o))
		}
		allErrs = append(allErrs, field.NotSupported(fldPath, optionType, supportedValues))
	}
	return allErrs
}

// ValidateOptionName validates the option's name and returns a list of errors.
func ValidateOptionName(name string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if !nameRegexp.MatchString(name) {
		detail := fmt.Sprintf("must match regex: %v", nameRegexp)
		allErrs = append(allErrs, field.Invalid(fldPath, name, detail))
	}
	return allErrs
}

// ValidateOptionBool validates the Bool option and returns a list of errors.
func ValidateOptionBool(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	sanitized := ZeroForNonConfig(option)
	sanitized.Bool = nil
	if !reflect.DeepEqual(execution.Option{}, sanitized) {
		allErrs = append(allErrs, NewForbiddenOptionConfigError(fldPath, option.Type))
	}
	allErrs = append(allErrs, ValidateBoolOptionConfig(option.Bool, fldPath.Child("bool"))...)
	if option.Required {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("required"), "bool option cannot be marked as required"))
	}
	return allErrs
}

// ValidateBoolOptionConfig validates the BoolOptionConfig and returns a list of errors.
func ValidateBoolOptionConfig(cfg *execution.BoolOptionConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if cfg == nil {
		cfg = &execution.BoolOptionConfig{}
	}
	if cfg.Format == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("format"), ""))
	} else if !cfg.Format.IsValid() {
		supportedValues := make([]string, 0, len(execution.BoolOptionFormatsAll))
		for _, format := range execution.BoolOptionFormatsAll {
			supportedValues = append(supportedValues, string(format))
		}
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("format"), cfg.Format, supportedValues))
	}
	return allErrs
}

// ValidateOptionString validates the String option and returns a list of errors.
func ValidateOptionString(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	sanitized := ZeroForNonConfig(option)
	sanitized.String = nil
	if !reflect.DeepEqual(execution.Option{}, sanitized) {
		allErrs = append(allErrs, NewForbiddenOptionConfigError(fldPath, option.Type))
	}
	return allErrs
}

// ValidateOptionSelect validates the Select option and returns a list of errors.
func ValidateOptionSelect(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	sanitized := ZeroForNonConfig(option)
	sanitized.Select = nil
	if !reflect.DeepEqual(execution.Option{}, sanitized) {
		allErrs = append(allErrs, NewForbiddenOptionConfigError(fldPath, option.Type))
	}
	allErrs = append(allErrs, ValidateSelectOptionConfig(option.Select, fldPath.Child("select"))...)
	return allErrs
}

// ValidateSelectOptionConfig validates the SelectOptionConfig and returns a list of errors.
func ValidateSelectOptionConfig(cfg *execution.SelectOptionConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if cfg == nil {
		cfg = &execution.SelectOptionConfig{}
	}
	if len(cfg.Values) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("values"), "at least 1 value is required"))
	}
	if len(cfg.Default) > 0 && !stringsutils.ContainsString(cfg.Values, cfg.Default) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("default"), cfg.Default, cfg.Values))
	}
	for i, value := range cfg.Values {
		if len(value) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("values").Index(i), ""))
		}
	}
	return allErrs
}

// ValidateOptionMulti validates the Multi option and returns a list of errors.
func ValidateOptionMulti(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	sanitized := ZeroForNonConfig(option)
	sanitized.Multi = nil
	if !reflect.DeepEqual(execution.Option{}, sanitized) {
		allErrs = append(allErrs, NewForbiddenOptionConfigError(fldPath, option.Type))
	}
	allErrs = append(allErrs, ValidateMultiOptionConfig(option.Multi, fldPath.Child("multi"))...)
	return allErrs
}

// ValidateMultiOptionConfig validates the MultiOptionConfig and returns a list of errors.
func ValidateMultiOptionConfig(cfg *execution.MultiOptionConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if cfg == nil {
		cfg = &execution.MultiOptionConfig{}
	}
	if len(cfg.Values) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("values"), "at least 1 value is required"))
	}
	for i, def := range cfg.Default {
		if !stringsutils.ContainsString(cfg.Values, def) {
			allErrs = append(allErrs, field.NotSupported(fldPath.Child("default").Index(i), def, cfg.Values))
		}
	}
	for i, value := range cfg.Values {
		if len(value) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("values").Index(i), ""))
		}
	}
	for i, value := range cfg.Default {
		if len(value) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("default").Index(i), ""))
		}
	}
	return allErrs
}

// ValidateOptionDate validates the Date option and returns a list of errors.
func ValidateOptionDate(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	sanitized := ZeroForNonConfig(option)
	sanitized.Date = nil
	if !reflect.DeepEqual(execution.Option{}, sanitized) {
		allErrs = append(allErrs, NewForbiddenOptionConfigError(fldPath, option.Type))
	}

	// NOTE(irvinlim): It seems that Moment.js will never throw errors when
	// encountering an invalid date format, this is probably because any invalid
	// tokens are simply treated as string literals.

	return allErrs
}
