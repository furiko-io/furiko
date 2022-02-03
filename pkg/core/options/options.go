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
	"regexp"
	"strings"
	"time"

	"github.com/nleeper/goment"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	stringsutils "github.com/furiko-io/furiko/pkg/utils/strings"
)

var (
	jobOptionNameRegexp = regexp.MustCompile(`^[a-zA-Z_0-9.-]+$`)
)

// MakeOptionVariableName returns the variable name for an option.
func MakeOptionVariableName(option execution.Option) string {
	return fmt.Sprintf("option.%v", option.Name)
}

// EvaluateOptions will evaluate all options in the given map.
func EvaluateOptions(
	options map[string]interface{},
	cfg *execution.OptionSpec,
	fldPath *field.Path,
) (map[string]string, field.ErrorList) {
	allErrs := field.ErrorList{}
	eval := make(map[string]string)
	if cfg == nil {
		return eval, nil
	}

	for _, option := range cfg.Options {
		// Evaluate option value, otherwise default to nil.
		var optionValue interface{}
		if val, ok := options[option.Name]; ok {
			optionValue = val
		}

		// Evaluate option using the value.
		optionStr, err := EvaluateOption(optionValue, option, fldPath.Key(option.Name))
		if err != nil {
			allErrs = append(allErrs, err)
			continue
		}
		eval[MakeOptionVariableName(option)] = optionStr
	}

	return eval, allErrs
}

// EvaluateOption will evaluate the value of the option against the given Option configuration.
// It assumes that the option is valid.
func EvaluateOption(value interface{}, option execution.Option, fldPath *field.Path) (string, *field.Error) {
	switch option.Type {
	case execution.OptionTypeBool:
		return EvaluateJobOptionBool(value, option.Bool, fldPath)
	case execution.OptionTypeString:
		return EvaluateJobOptionString(value, option, option.String, fldPath)
	case execution.OptionTypeSelect:
		return EvaluateJobOptionSelect(value, option, option.Select, fldPath)
	case execution.OptionTypeMulti:
		return EvaluateJobOptionMulti(value, option, option.Multi, fldPath)
	case execution.OptionTypeDate:
		return EvaluateJobOptionDate(value, option, option.Date, fldPath)
	}
	return "", field.Invalid(fldPath, value, fmt.Sprintf("unsupported option type %v", option.Type))
}

func EvaluateJobOptionBool(
	value interface{}, cfg *execution.BoolOptionConfig, fldPath *field.Path,
) (string, *field.Error) {
	if cfg == nil {
		cfg = &execution.BoolOptionConfig{}
	}
	if value == nil {
		value = cfg.Default
	}
	v, ok := value.(bool)
	if !ok {
		return "", field.Invalid(fldPath, value, fmt.Sprintf("expected bool, got %T", value))
	}
	result, err := cfg.FormatValue(v)
	if err != nil {
		return "", field.Invalid(fldPath, v, err.Error())
	}
	return result, nil
}

func EvaluateJobOptionString(
	value interface{}, option execution.Option, cfg *execution.StringOptionConfig, fldPath *field.Path,
) (string, *field.Error) {
	if cfg == nil {
		cfg = &execution.StringOptionConfig{}
	}
	if value == nil {
		value = cfg.Default
	}
	v, ok := value.(string)
	if !ok {
		return "", field.Invalid(fldPath, value, fmt.Sprintf("expected string, got %T", value))
	}
	if cfg.TrimSpaces {
		v = strings.TrimSpace(v)
	}
	if option.Required && len(v) == 0 {
		return "", field.Required(fldPath, "option is required")
	}
	return v, nil
}

func EvaluateJobOptionSelect(
	value interface{}, option execution.Option, cfg *execution.SelectOptionConfig, fldPath *field.Path,
) (string, *field.Error) {
	if cfg == nil {
		cfg = &execution.SelectOptionConfig{}
	}
	if value == nil {
		value = cfg.Default
	}
	v, ok := value.(string)
	if !ok {
		return "", field.Invalid(fldPath, value, fmt.Sprintf("expected string, got %T", value))
	}
	if len(v) > 0 && !cfg.AllowCustom && !stringsutils.ContainsString(cfg.Values, v) {
		return "", field.NotSupported(fldPath, v, cfg.Values)
	}
	if len(v) == 0 && option.Required {
		return "", field.Required(fldPath, "option is required")
	}
	return v, nil
}

func EvaluateJobOptionMulti(
	value interface{}, option execution.Option, cfg *execution.MultiOptionConfig, fldPath *field.Path,
) (string, *field.Error) {
	if cfg == nil {
		cfg = &execution.MultiOptionConfig{}
	}

	// Check if nil
	if value == nil {
		value = []string{}
	}

	// Convert []interface{} to []string if necessary
	if vi, ok := value.([]interface{}); ok {
		newValue := make([]string, 0, len(vi))
		for _, vx := range vi {
			v, ok := vx.(string)
			if !ok {
				return "", field.Invalid(fldPath, vx, fmt.Sprintf("expected string, got %T", value))
			}
			newValue = append(newValue, v)
		}
		value = newValue
	}

	// Type assertion for []string
	v, ok := value.([]string)
	if !ok {
		return "", field.Invalid(fldPath, value, fmt.Sprintf("expected []string, got %T", value))
	}

	// Set default if empty
	if len(v) == 0 {
		v = cfg.Default
	}

	if len(v) == 0 && option.Required {
		return "", field.Required(fldPath, "option is required")
	}

	// Validate AllowCustom.
	for _, val := range v {
		if len(v) > 0 && !cfg.AllowCustom {
			if !stringsutils.ContainsString(cfg.Values, val) {
				return "", field.NotSupported(fldPath, val, cfg.Values)
			}
		}
		if len(val) == 0 {
			return "", field.Required(fldPath, "")
		}
	}

	return strings.Join(v, cfg.Delimiter), nil
}

func EvaluateJobOptionDate(
	value interface{},
	option execution.Option,
	cfg *execution.DateOptionConfig,
	fldPath *field.Path,
) (string, *field.Error) {
	if cfg == nil {
		cfg = &execution.DateOptionConfig{}
	}

	if value == nil {
		value = ""
	}

	var t time.Time
	switch v := value.(type) {
	case *time.Time:
		if v != nil {
			t = *v
		}
	case time.Time:
		t = v
	case string:
		if len(v) > 0 {
			parsed, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return "", field.Invalid(fldPath, value, fmt.Sprintf("cannot parse time: %v", err))
			}
			t = parsed
		}
	default:
		return "", field.Invalid(fldPath, value, fmt.Sprintf("expected string, got %T", value))
	}

	if t.IsZero() {
		if option.Required {
			return "", field.Required(fldPath, "option is required")
		}
		return "", nil
	}

	formatted, err := FormatAsMoment(t, cfg.Format)
	if err != nil {
		return "", field.Invalid(fldPath, value, err.Error())
	}
	return formatted, nil
}

func FormatAsMoment(ts time.Time, format string) (string, error) {
	g, err := goment.New(ts)
	if err != nil {
		return "", errors.Wrapf(err, "invalid time")
	}
	var args []interface{}
	if len(format) > 0 {
		args = append(args, format)
	}
	return g.Format(args...), nil
}

// EvaluateOptionDefault evaluates the option and returns the default value.
func EvaluateOptionDefault(option execution.Option) (string, error) {
	switch option.Type {
	case execution.OptionTypeBool:
		return EvaluateJobOptionDefaultBool(option.Bool)
	case execution.OptionTypeString:
		return EvaluateJobOptionDefaultString(option.String)
	case execution.OptionTypeSelect:
		return EvaluateJobOptionDefaultSelect(option.Select)
	case execution.OptionTypeMulti:
		return EvaluateJobOptionDefaultMulti(option.Multi)
	case execution.OptionTypeDate:
		return "", nil
	}
	return "", fmt.Errorf(`unsupported option type "%v"`, option.Type)
}

func EvaluateJobOptionDefaultBool(cfg *execution.BoolOptionConfig) (string, error) {
	if cfg == nil {
		cfg = &execution.BoolOptionConfig{}
	}
	return cfg.FormatValue(cfg.Default)
}

func EvaluateJobOptionDefaultString(cfg *execution.StringOptionConfig) (string, error) {
	if cfg == nil {
		cfg = &execution.StringOptionConfig{}
	}
	value := cfg.Default
	if cfg.TrimSpaces {
		value = strings.TrimSpace(value)
	}
	return value, nil
}

func EvaluateJobOptionDefaultSelect(cfg *execution.SelectOptionConfig) (string, error) {
	if cfg == nil {
		cfg = &execution.SelectOptionConfig{}
	}
	return cfg.Default, nil
}

func EvaluateJobOptionDefaultMulti(cfg *execution.MultiOptionConfig) (string, error) {
	if cfg == nil {
		cfg = &execution.MultiOptionConfig{}
	}
	return strings.Join(cfg.Default, cfg.Delimiter), nil
}

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
			allErrs = append(allErrs, ValidateJobOption(option, fldPath)...)
		}
	}
	return allErrs
}

func ValidateJobOption(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateJobOptionType(option.Type, fldPath)...)
	allErrs = append(allErrs, ValidateJobOptionName(option.Name, fldPath)...)
	switch option.Type {
	case execution.OptionTypeBool:
		allErrs = append(allErrs, ValidateJobOptionBool(option, fldPath)...)
	case execution.OptionTypeString:
		allErrs = append(allErrs, ValidateJobOptionString(option, fldPath)...)
	case execution.OptionTypeSelect:
		allErrs = append(allErrs, ValidateJobOptionSelect(option, fldPath)...)
	case execution.OptionTypeMulti:
		allErrs = append(allErrs, ValidateJobOptionMulti(option, fldPath)...)
	case execution.OptionTypeDate:
		allErrs = append(allErrs, ValidateJobOptionDate(option, fldPath)...)
	}
	return allErrs
}

func ValidateJobOptionType(optionType execution.OptionType, fldPath *field.Path) field.ErrorList {
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

func ValidateJobOptionName(name string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if !jobOptionNameRegexp.MatchString(name) {
		detail := fmt.Sprintf("must match regex: %v", jobOptionNameRegexp)
		allErrs = append(allErrs, field.Invalid(fldPath, name, detail))
	}
	return allErrs
}

func ValidateJobOptionBool(option execution.Option, fldPath *field.Path) field.ErrorList {
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

func ValidateJobOptionString(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	sanitized := ZeroForNonConfig(option)
	sanitized.String = nil
	if !reflect.DeepEqual(execution.Option{}, sanitized) {
		allErrs = append(allErrs, NewForbiddenOptionConfigError(fldPath, option.Type))
	}
	return allErrs
}

func ValidateJobOptionSelect(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	sanitized := ZeroForNonConfig(option)
	sanitized.Select = nil
	if !reflect.DeepEqual(execution.Option{}, sanitized) {
		allErrs = append(allErrs, NewForbiddenOptionConfigError(fldPath, option.Type))
	}
	allErrs = append(allErrs, ValidateSelectOptionConfig(option.Select, fldPath.Child("select"))...)
	return allErrs
}

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

func ValidateJobOptionMulti(option execution.Option, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	sanitized := ZeroForNonConfig(option)
	sanitized.Multi = nil
	if !reflect.DeepEqual(execution.Option{}, sanitized) {
		allErrs = append(allErrs, NewForbiddenOptionConfigError(fldPath, option.Type))
	}
	allErrs = append(allErrs, ValidateMultiOptionConfig(option.Multi, fldPath.Child("multi"))...)
	return allErrs
}

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

func ValidateJobOptionDate(option execution.Option, fldPath *field.Path) field.ErrorList {
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

// DefaultOptionSpec populates default values for a OptionSpec.
func DefaultOptionSpec(spec *execution.OptionSpec) *execution.OptionSpec {
	newSpec := spec.DeepCopy()
	for i, option := range spec.Options {
		newSpec.Options[i] = DefaultJobOption(option)
	}
	return newSpec
}

func DefaultJobOption(option execution.Option) execution.Option {
	switch option.Type {
	case execution.OptionTypeBool:
		return DefaultJobOptionBool(option)
	}
	return option
}

func DefaultJobOptionBool(option execution.Option) execution.Option {
	newOption := option.DeepCopy()
	if newOption.Bool == nil {
		newOption.Bool = &execution.BoolOptionConfig{}
	}
	if newOption.Bool.Format == "" {
		newOption.Bool.Format = execution.BoolOptionFormatTrueFalse
	}
	return *newOption
}
