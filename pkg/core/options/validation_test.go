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
	"testing"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
)

func TestValidateOptionSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    *execution.OptionSpec
		wantErr bool
	}{
		{
			name: "nil config",
			spec: nil,
		},
		{
			name: "no options",
			spec: &execution.OptionSpec{
				Options: nil,
			},
		},
		{
			name: "duplicate option names",
			spec: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Name:   "option",
						Type:   execution.OptionTypeString,
						String: &execution.StringOptionConfig{},
					},
					{
						Name: "option",
						Type: execution.OptionTypeSelect,
						Select: &execution.SelectOptionConfig{
							Values: []string{"a", "b"},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := options.ValidateOptionSpec(tt.spec, rootPath).ToAggregate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOptionSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJobOption(t *testing.T) {
	tests := []struct {
		name    string
		option  execution.Option
		wantErr bool
	}{
		{
			name: "invalid type",
			option: execution.Option{
				Type: "__invalid__",
				Name: "opt",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			option: execution.Option{
				Type:   execution.OptionTypeString,
				String: &execution.StringOptionConfig{},
			},
			wantErr: true,
		},
		{
			name: "name has spaces",
			option: execution.Option{
				Type:   execution.OptionTypeString,
				Name:   "option name",
				String: &execution.StringOptionConfig{},
			},
			wantErr: true,
		},
		{
			name: "name has unallowed characters",
			option: execution.Option{
				Type:   execution.OptionTypeString,
				Name:   "<option_name>",
				String: &execution.StringOptionConfig{},
			},
			wantErr: true,
		},
		{
			name: "bool option",
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Default: true,
					Format:  execution.BoolOptionFormatTrueFalse,
				},
			},
		},
		{
			name: "bool option missing fields",
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{},
			},
			wantErr: true,
		},
		{
			name: "bool option invalid format",
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Format: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "bool option custom format empty values ok",
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Format: execution.BoolOptionFormatCustom,
				},
			},
		},
		{
			name: "bool option cannot be required",
			option: execution.Option{
				Type:     execution.OptionTypeBool,
				Name:     "opt",
				Required: true,
				Bool: &execution.BoolOptionConfig{
					Default: true,
					Format:  execution.BoolOptionFormatTrueFalse,
				},
			},
			wantErr: true,
		},
		{
			name: "bool option empty config",
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
			},
			wantErr: true,
		},
		{
			name: "bool option with non-bool config",
			option: execution.Option{
				Type: execution.OptionTypeBool,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Format: execution.BoolOptionFormatTrueFalse,
				},
				String: &execution.StringOptionConfig{},
			},
			wantErr: true,
		},
		{
			name: "string option",
			option: execution.Option{
				Type: execution.OptionTypeString,
				Name: "opt",
				String: &execution.StringOptionConfig{
					Default:    "hello",
					TrimSpaces: true,
				},
			},
		},
		{
			name: "string option require default value",
			option: execution.Option{
				Type:     execution.OptionTypeString,
				Name:     "opt",
				Required: true,
				String:   &execution.StringOptionConfig{},
			},
		},
		{
			name: "string option require default value after trimming",
			option: execution.Option{
				Type:     execution.OptionTypeString,
				Name:     "opt",
				Required: true,
				String: &execution.StringOptionConfig{
					Default:    " ",
					TrimSpaces: true,
				},
			},
		},
		{
			name: "string option empty config",
			option: execution.Option{
				Type: execution.OptionTypeString,
				Name: "opt",
			},
		},
		{
			name: "string option with non-string config",
			option: execution.Option{
				Type: execution.OptionTypeString,
				Name: "opt",
				Bool: &execution.BoolOptionConfig{
					Format: execution.BoolOptionFormatTrueFalse,
				},
			},
			wantErr: true,
		},
		{
			name: "select option",
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Default:     "a",
					Values:      []string{"a", "b"},
					AllowCustom: true,
				},
			},
		},
		{
			name: "select option empty config",
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
			},
			wantErr: true,
		},
		{
			name: "select option with non-select config",
			option: execution.Option{
				Type: execution.OptionTypeSelect,
				Name: "opt",
				Select: &execution.SelectOptionConfig{
					Default:     "a",
					Values:      []string{"a", "b"},
					AllowCustom: true,
				},
				String: &execution.StringOptionConfig{},
			},
			wantErr: true,
		},
		{
			name: "select option missing values",
			option: execution.Option{
				Type:     execution.OptionTypeSelect,
				Name:     "opt",
				Required: true,
				Select:   &execution.SelectOptionConfig{},
			},
			wantErr: true,
		},
		{
			name: "select option don't require default value",
			option: execution.Option{
				Type:     execution.OptionTypeSelect,
				Name:     "opt",
				Required: true,
				Select: &execution.SelectOptionConfig{
					Values: []string{"a", "b"},
				},
			},
			wantErr: false,
		},
		{
			name: "select option cannot be empty",
			option: execution.Option{
				Type:     execution.OptionTypeSelect,
				Name:     "opt",
				Required: true,
				Select: &execution.SelectOptionConfig{
					Values:  []string{"a", "b", ""},
					Default: "a",
				},
			},
			wantErr: true,
		},
		{
			name: "select option default not in allowed values",
			option: execution.Option{
				Type:     execution.OptionTypeSelect,
				Name:     "opt",
				Required: true,
				Select: &execution.SelectOptionConfig{
					Default:     "c",
					Values:      []string{"a", "b"},
					AllowCustom: true,
				},
			},
			wantErr: true,
		},
		{
			name: "multi option",
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Delimiter: ",",
					Values:    []string{"a", "b"},
					Default:   []string{"a"},
				},
			},
		},
		{
			name: "multi option default value not in allowed values",
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Delimiter:   ",",
					Values:      []string{"a", "b"},
					Default:     []string{"b", "c"},
					AllowCustom: true,
				},
			},
			wantErr: true,
		},
		{
			name: "multi option empty config",
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
			},
			wantErr: true,
		},
		{
			name: "multi option with non-multi config",
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Delimiter: ",",
					Values:    []string{"a", "b"},
					Default:   []string{"a"},
				},
				String: &execution.StringOptionConfig{},
			},
			wantErr: true,
		},
		{
			name: "multi option empty delimiter",
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Values: []string{"a", "b"},
				},
			},
		},
		{
			name: "multi option missing values",
			option: execution.Option{
				Type: execution.OptionTypeMulti,
				Name: "opt",
				Multi: &execution.MultiOptionConfig{
					Delimiter: ",",
				},
			},
			wantErr: true,
		},
		{
			name: "multi option don't require default",
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Delimiter: ",",
					Values:    []string{"a", "b"},
				},
			},
			wantErr: false,
		},
		{
			name: "multi option default values not in allowed values",
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Delimiter: ",",
					Default:   []string{"b", "c"},
					Values:    []string{"a", "b"},
				},
			},
			wantErr: true,
		},
		{
			name: "multi option cannot be empty",
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Delimiter: ",",
					Default:   []string{"b"},
					Values:    []string{"a", "b", ""},
				},
			},
			wantErr: true,
		},
		{
			name: "multi default value cannot be empty",
			option: execution.Option{
				Type:     execution.OptionTypeMulti,
				Name:     "opt",
				Required: true,
				Multi: &execution.MultiOptionConfig{
					Delimiter: ",",
					Default:   []string{""},
					Values:    []string{"a", "b"},
				},
			},
			wantErr: true,
		},
		{
			name: "date option",
			option: execution.Option{
				Type: execution.OptionTypeDate,
				Name: "opt",
				Date: &execution.DateOptionConfig{
					Format: "YYYY-MM-DD HH:mm:ss",
				},
			},
		},
		{
			name: "date option empty config",
			option: execution.Option{
				Type: execution.OptionTypeDate,
				Name: "opt",
			},
		},
		{
			name: "date option with non-date config",
			option: execution.Option{
				Type:   execution.OptionTypeDate,
				Name:   "opt",
				String: &execution.StringOptionConfig{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := options.ValidateOption(tt.option, rootPath).ToAggregate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOption() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
