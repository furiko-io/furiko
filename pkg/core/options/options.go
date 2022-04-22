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
	"time"

	"github.com/nleeper/goment"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	stringsutils "github.com/furiko-io/furiko/pkg/utils/strings"
)

var (
	nameRegexp = regexp.MustCompile(`^[a-zA-Z_0-9.-]+$`)
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
func EvaluateOption(value interface{}, option execution.Option, fldPath *field.Path) (string, *field.Error) {
	switch option.Type {
	case execution.OptionTypeBool:
		return EvaluateOptionBool(value, option.Bool, fldPath)
	case execution.OptionTypeString:
		return EvaluateOptionString(value, option, option.String, fldPath)
	case execution.OptionTypeSelect:
		return EvaluateOptionSelect(value, option, option.Select, fldPath)
	case execution.OptionTypeMulti:
		return EvaluateOptionMulti(value, option, option.Multi, fldPath)
	case execution.OptionTypeDate:
		return EvaluateOptionDate(value, option, option.Date, fldPath)
	}
	return "", field.Invalid(fldPath, value, fmt.Sprintf("unsupported option type %v", option.Type))
}

// EvaluateOptionBool evaluates the value of the Bool option.
func EvaluateOptionBool(
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

// EvaluateOptionString evaluates the value of the String option.
func EvaluateOptionString(
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

// EvaluateOptionSelect evaluates the value of the Select option.
func EvaluateOptionSelect(
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

// EvaluateOptionMulti evaluates the value of the Multi option.
func EvaluateOptionMulti(
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

// EvaluateOptionDate evaluates the value of the Date option.
func EvaluateOptionDate(
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

// FormatAsMoment formats the given time.Time with the moment.js format.
// The time.Location of ts will be used for formatting the given time.
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
