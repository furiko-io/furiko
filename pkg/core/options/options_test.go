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

package options_test

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/validation/field"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
)

const (
	mockTime = "2021-02-09T12:06:09+08:00"
)

var (
	stdTime, _ = time.Parse(time.RFC3339, mockTime)
	rootPath   = field.NewPath("root")
)

func TestEvaluateOption(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		option  execution.Option
		want    string
		wantErr bool
	}{
		{
			name:  "bool option, nil value",
			value: nil,
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Format: execution.BoolOptionFormatTrueFalse,
				},
			},
			want: "false",
		},
		{
			name:  "bool option, nil value with a default",
			value: nil,
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Default: true,
					Format:  execution.BoolOptionFormatTrueFalse,
				},
			},
			want: "true",
		},
		{
			name:  "bool option, nil value with custom format",
			value: nil,
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Default: true,
					Format:  execution.BoolOptionFormatOneZero,
				},
			},
			want: "1",
		},
		{
			name:  "bool option specify false",
			value: false,
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Default: true,
					Format:  execution.BoolOptionFormatYesNo,
				},
			},
			want: "no",
		},
		{
			name:  "bool option specify true",
			value: true,
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Default: false,
					Format:  execution.BoolOptionFormatYesNo,
				},
			},
			want: "yes",
		},
		{
			name:  "bool option custom format true",
			value: true,
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Format:  execution.BoolOptionFormatCustom,
					TrueVal: "--verbose",
				},
			},
			want: "--verbose",
		},
		{
			name:  "bool option custom format false",
			value: false,
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Format:  execution.BoolOptionFormatCustom,
					TrueVal: "--verbose",
				},
			},
			want: "",
		},
		{
			name:  "bool option, not a bool",
			value: "true",
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Format: execution.BoolOptionFormatTrueFalse,
				},
			},
			wantErr: true,
		},
		{
			name:  "bool option, got empty string",
			value: "",
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Format: execution.BoolOptionFormatTrueFalse,
				},
			},
			wantErr: true,
		},
		{
			name:  "string option, nil value",
			value: nil,
			option: execution.Option{
				Type:   execution.OptionTypeString,
				Name:   "opt",
				String: &execution.StringOptionConfig{},
			},
			want: "",
		},
		{
			name:  "string option, empty string",
			value: "",
			option: execution.Option{
				Type:   execution.OptionTypeString,
				Name:   "opt",
				String: &execution.StringOptionConfig{},
			},
			want: "",
		},
		{
			name:  "string option, specify value",
			value: "hello",
			option: execution.Option{
				Type:   execution.OptionTypeString,
				Name:   "opt",
				String: &execution.StringOptionConfig{},
			},
			want: "hello",
		},
		{
			name:  "string option, specify value with default",
			value: " world ",
			option: execution.Option{
				Type:     execution.OptionTypeString,
				Name:     "opt",
				Required: true,
				String: &execution.StringOptionConfig{
					Default: "hello",
				},
			},
			want: " world ",
		},
		{
			name:  "string option, specify value with default, trim spaces",
			value: " world ",
			option: execution.Option{
				Type:     execution.OptionTypeString,
				Name:     "opt",
				Required: true,
				String: &execution.StringOptionConfig{
					Default:    "hello",
					TrimSpaces: true,
				},
			},
			want: "world",
		},
		{
			name:  "string option, nil value, required, with default",
			value: nil,
			option: execution.Option{
				Type:     execution.OptionTypeString,
				Name:     "opt",
				Required: true,
				String: &execution.StringOptionConfig{
					Default: "hello ",
				},
			},
			want: "hello ",
		},
		{
			name:  "string option, nil value, required, no default",
			value: nil,
			option: execution.Option{
				Type:     execution.OptionTypeString,
				Name:     "opt",
				Required: true,
			},
			wantErr: true,
		},
		{
			name:  "string option, nil value, not required, no default",
			value: nil,
			option: execution.Option{
				Type: execution.OptionTypeString,
				Name: "opt",
			},
			want: "",
		},
		{
			name:  "string option, nil value, not required, with default",
			value: nil,
			option: execution.Option{
				Type: execution.OptionTypeString,
				Name: "opt",
				String: &execution.StringOptionConfig{
					Default: "default",
				},
			},
			want: "default",
		},
		{
			name:  "string option, nil value, trim default value",
			value: nil,
			option: execution.Option{
				Type:     execution.OptionTypeString,
				Name:     "opt",
				Required: true,
				String: &execution.StringOptionConfig{
					Default:    "hello ",
					TrimSpaces: true,
				},
			},
			want: "hello",
		},
		{
			name:  "string option, not a string",
			value: true,
			option: execution.Option{
				Type: execution.OptionTypeString,
				Name: "opt",
			},
			wantErr: true,
		},
		{
			name:  "string option, empty string, required, no default",
			value: "",
			option: execution.Option{
				Type:     execution.OptionTypeString,
				Name:     "opt",
				Required: true,
			},
			wantErr: true,
		},
		{
			name:  "string option, empty string, required, with default",
			value: "",
			option: execution.Option{
				Type:     execution.OptionTypeString,
				Name:     "opt",
				Required: true,
				String: &execution.StringOptionConfig{
					Default: "hello",
				},
			},
			wantErr: true,
		},
		{
			name:  "string option, empty string, not required, no default",
			value: "",
			option: execution.Option{
				Type: execution.OptionTypeString,
				Name: "opt",
			},
			want: "",
		},
		{
			name:  "string option, empty string, not required, with default",
			value: "",
			option: execution.Option{
				Type: execution.OptionTypeString,
				Name: "opt",
				String: &execution.StringOptionConfig{
					Default: "hello",
				},
			},
			want: "",
		},
		{
			name:  "select option, nil value",
			value: nil,
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Values: []string{"a", "b"},
				},
			},
			want: "",
		},
		{
			name:  "select option, specify value",
			value: "a",
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Values: []string{"a", "b"},
				},
			},
			want: "a",
		},
		{
			name:  "select option, allow empty string when not required",
			value: "",
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Values:  []string{"a", "b"},
					Default: "a",
				},
			},
			want: "",
		},
		{
			name:  "select option, no allow custom",
			value: "c",
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Values: []string{"a", "b"},
				},
			},
			wantErr: true,
		},
		{
			name:  "select option, allow custom",
			value: "c",
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Values:      []string{"a", "b"},
					AllowCustom: true,
				},
			},
			want: "c",
		},
		{
			name:  "select option, nil value with a default",
			value: nil,
			option: execution.Option{
				Type:     execution.OptionTypeSelect,
				Name:     "opt",
				Required: true,
				Select: &execution.SelectOptionConfig{
					Default:     "b",
					Values:      []string{"a", "b"},
					AllowCustom: true,
				},
			},
			want: "b",
		},
		{
			name:  "select option, allow empty string when not required and allow custom",
			value: "",
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Default:     "a",
					Values:      []string{"a", "b"},
					AllowCustom: true,
				},
			},
			want: "",
		},
		{
			name:  "select option, allow empty string when not required",
			value: "",
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Default: "a",
					Values:  []string{"a", "b"},
				},
			},
			want: "",
		},
		{
			name:  "select option, empty string cannot be used when required",
			value: "",
			option: execution.Option{
				Type:     execution.OptionTypeSelect,
				Name:     "opt",
				Required: true,
				Select: &execution.SelectOptionConfig{
					Default:     "b",
					Values:      []string{"a", "b"},
					AllowCustom: true,
				},
			},
			wantErr: true,
		},
		{
			name:  "select option, specify value with a default",
			value: "a",
			option: execution.Option{
				Type:     execution.OptionTypeSelect,
				Name:     "opt",
				Required: true,
				Select: &execution.SelectOptionConfig{
					Default:     "b",
					Values:      []string{"a", "b"},
					AllowCustom: true,
				},
			},
			want: "a",
		},
		{
			name:  "select option, not a string",
			value: true,
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Values: []string{"a", "b"},
				},
			},
			wantErr: true,
		},
		{
			name:  "multi option, nil value",
			value: nil,
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:    []string{"a", "b"},
					Delimiter: ",",
				},
			},
			want: "",
		},
		{
			name:  "multi option, nil []interface{} value",
			value: []interface{}(nil),
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:    []string{"a", "b"},
					Delimiter: ",",
				},
			},
			want: "",
		},
		{
			name:  "multi option, specify value",
			value: []string{"a"},
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:    []string{"a", "b"},
					Delimiter: ",",
				},
			},
			want: "a",
		},
		{
			name:  "multi option, specify []interface{} value",
			value: []interface{}{"a"},
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:    []string{"a", "b"},
					Delimiter: ",",
				},
			},
			want: "a",
		},
		{
			name:  "multi option, specify multiple value",
			value: []string{"a", "b"},
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:    []string{"a", "b"},
					Delimiter: ",",
				},
			},
			want: "a,b",
		},
		{
			name:  "multi option, no allow custom",
			value: []string{"c"},
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:    []string{"a", "b"},
					Delimiter: ",",
				},
			},
			wantErr: true,
		},
		{
			name:  "multi option, allow custom",
			value: []string{"c"},
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:      []string{"a", "b"},
					Delimiter:   ",",
					AllowCustom: true,
				},
			},
			want: "c",
		},
		{
			name:  "multi option, nil value with a default",
			value: nil,
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Default:     []string{"b"},
					Values:      []string{"a", "b"},
					Delimiter:   ",",
					AllowCustom: true,
				},
			},
			want: "b",
		},
		{
			name:  "multi option, nil []interface{} value with a default",
			value: []interface{}(nil),
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Default:     []string{"b"},
					Values:      []string{"a", "b"},
					Delimiter:   ",",
					AllowCustom: true,
				},
			},
			want: "b",
		},
		{
			name:  "multi option, empty []interface{} value with a default",
			value: []interface{}{},
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Values:      []string{"a", "b"},
					Default:     []string{"a"},
					Delimiter:   ",",
					AllowCustom: true,
				},
			},
			want: "a",
		},
		{
			name:  "multi option, nil []interface{} value without a default",
			value: []interface{}(nil),
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Values:      []string{"a", "b"},
					Delimiter:   ",",
					AllowCustom: true,
				},
			},
			wantErr: true,
		},
		{
			name:  "multi option, empty []interface{} value without a default",
			value: []interface{}{},
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Values:      []string{"a", "b"},
					Delimiter:   ",",
					AllowCustom: true,
				},
			},
			wantErr: true,
		},
		{
			name:  "multi option, specify value with a default",
			value: []string{"a"},
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Default:     []string{"b"},
					Values:      []string{"a", "b"},
					Delimiter:   ",",
					AllowCustom: true,
				},
			},
			want: "a",
		},
		{
			name:  "multi option, not a []string or []interface{}",
			value: "a",
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:    []string{"a", "b"},
					Delimiter: ",",
				},
			},
			wantErr: true,
		},
		{
			name:  "multi option, value is empty string",
			value: "",
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:    []string{"a", "b"},
					Delimiter: ",",
				},
			},
			wantErr: true,
		},
		{
			name:  "multi option, []interface{} contains non-string",
			value: []interface{}{"a", 2},
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:      []string{"a", "b"},
					Delimiter:   ",",
					AllowCustom: true,
				},
			},
			wantErr: true,
		},
		{
			name:  "multi option, []interface{} contains empty string",
			value: []interface{}{"a", ""},
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values:      []string{"a", "b"},
					Delimiter:   ",",
					AllowCustom: true,
				},
			},
			wantErr: true,
		},
		{
			name:  "date option, nil value",
			value: nil,
			option: execution.Option{
				Type: execution.OptionTypeDate,
				Name: "opt",
			},
			want: "",
		},
		{
			name: "date option, required",
			option: execution.Option{
				Type:     execution.OptionTypeDate,
				Name:     "opt",
				Required: true,
			},
			wantErr: true,
		},
		{
			name:  "date option with RFC3339 string",
			value: stdTime.Format(time.RFC3339),
			option: execution.Option{
				Type: execution.OptionTypeDate,
				Name: "opt",
			},
			want: stdTime.Format(time.RFC3339),
		},
		{
			name:  "date option with invalid string",
			value: "invalid",
			option: execution.Option{
				Type: execution.OptionTypeDate,
				Name: "opt",
			},
			wantErr: true,
		},
		{
			name:  "date option with time.Time",
			value: stdTime,
			option: execution.Option{
				Type: execution.OptionTypeDate,
				Name: "opt",
			},
			want: stdTime.Format(time.RFC3339),
		},
		{
			name:  "date option with *time.Time",
			value: &stdTime,
			option: execution.Option{
				Type: execution.OptionTypeDate,
				Name: "opt",
			},
			want: stdTime.Format(time.RFC3339),
		},
		{
			name:  "date option, custom format",
			value: stdTime.Format(time.RFC3339),
			option: execution.Option{
				Type: execution.OptionTypeDate,
				Name: "opt",
				Date: &execution.DateOptionConfig{
					Format: "D MMM YYYY",
				},
			},
			want: "9 Feb 2021",
		},
		{
			name:  "date option with number input, already deprecated",
			value: stdTime.Unix(),
			option: execution.Option{
				Type: execution.OptionTypeDate,
				Name: "opt",
				Date: &execution.DateOptionConfig{
					Format: "D MMM YYYY",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate the option first, as EvaluateOption assumes a valid option.
			if err := options.ValidateOption(tt.option, rootPath).ToAggregate(); err != nil {
				t.Errorf("ValidateOption() got error %v", err)
				return
			}

			// Evaluate the option.
			got, err := options.EvaluateOption(tt.value, tt.option, rootPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateOption() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateOption() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateOptions(t *testing.T) {
	type args struct {
		options map[string]interface{}
		cfg     *execution.OptionSpec
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "no options to evaluate",
			args: args{},
			want: map[string]string{},
		},
		{
			name: "evaluate options with nil config",
			args: args{
				options: map[string]interface{}{},
				cfg: &execution.OptionSpec{
					Options: []execution.Option{
						{
							Name: "my_option",
							Type: execution.OptionTypeString,
						},
						{
							Name: "my_select_option",
							Type: execution.OptionTypeSelect,
						},
					},
				},
			},
			want: map[string]string{
				"option.my_option":        "",
				"option.my_select_option": "",
			},
		},
		{
			name: "evaluate empty options with default",
			args: args{
				options: map[string]interface{}{},
				cfg: &execution.OptionSpec{
					Options: []execution.Option{
						{
							Name: "my_option",
							Type: execution.OptionTypeString,
							String: &execution.StringOptionConfig{
								Default: "default_value",
							},
						},
						{
							Name: "my_select_option",
							Type: execution.OptionTypeSelect,
							Select: &execution.SelectOptionConfig{
								Values: []string{"a", "b", "c"},
							},
						},
					},
				},
			},
			want: map[string]string{
				"option.my_option":        "default_value",
				"option.my_select_option": "",
			},
		},
		{
			name: "evaluate empty options with required",
			args: args{
				options: map[string]interface{}{},
				cfg: &execution.OptionSpec{
					Options: []execution.Option{
						{
							Name:     "my_option",
							Type:     execution.OptionTypeString,
							Required: true,
							String:   &execution.StringOptionConfig{},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "evaluate override options with default",
			args: args{
				options: map[string]interface{}{
					"my_option": "override_value",
				},
				cfg: &execution.OptionSpec{
					Options: []execution.Option{
						{
							Name:     "my_option",
							Type:     execution.OptionTypeString,
							Required: true,
							String: &execution.StringOptionConfig{
								Default: "default_value",
							},
						},
						{
							Name: "my_select_option",
							Type: execution.OptionTypeSelect,
							Select: &execution.SelectOptionConfig{
								Values:  []string{"a", "b", "c"},
								Default: "a",
							},
						},
					},
				},
			},
			want: map[string]string{
				"option.my_option":        "override_value",
				"option.my_select_option": "a",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := options.EvaluateOptions(tt.args.options, tt.args.cfg, rootPath)
			if (errs.ToAggregate() != nil) != tt.wantErr {
				t.Errorf("EvaluateOptions() error = %v, wantErr %v", errs, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EvaluateOptions() got = %v, want %v", got, tt.want)
			}
		})
	}
}
