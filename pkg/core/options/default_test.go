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

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
)

func TestMakeDefaultOptions(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *execution.OptionSpec
		want    map[string]string
		wantErr bool
	}{
		{
			name: "nil options config",
			cfg:  nil,
			want: map[string]string{},
		},
		{
			name: "empty list of options",
			cfg:  &execution.OptionSpec{},
			want: map[string]string{},
		},
		{
			name: "bool without default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeBool,
						Name: "boolValue",
						Bool: &execution.BoolOptionConfig{
							Format: execution.BoolOptionFormatTrueFalse,
						},
					},
				},
			},
			want: map[string]string{
				"option.boolValue": "false",
			},
		},
		{
			name: "bool with default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeBool,
						Name: "boolValue",
						Bool: &execution.BoolOptionConfig{
							Format:  execution.BoolOptionFormatTrueFalse,
							Default: true,
						},
					},
				},
			},
			want: map[string]string{
				"option.boolValue": "true",
			},
		},
		{
			name: "bool with custom format",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeBool,
						Name: "boolValue",
						Bool: &execution.BoolOptionConfig{
							Format:  execution.BoolOptionFormatCustom,
							TrueVal: "--dry-run",
						},
					},
				},
			},
			want: map[string]string{
				"option.boolValue": "",
			},
		},
		{
			name: "bool with custom format with default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeBool,
						Name: "boolValue",
						Bool: &execution.BoolOptionConfig{
							Format:  execution.BoolOptionFormatCustom,
							TrueVal: "--dry-run",
							Default: true,
						},
					},
				},
			},
			want: map[string]string{
				"option.boolValue": "--dry-run",
			},
		},
		{
			name: "string without default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type:   execution.OptionTypeString,
						Name:   "stringval",
						String: &execution.StringOptionConfig{},
					},
				},
			},
			want: map[string]string{
				"option.stringval": "",
			},
		},
		{
			name: "string with default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "stringval",
						String: &execution.StringOptionConfig{
							Default: "hello ",
						},
					},
				},
			},
			want: map[string]string{
				"option.stringval": "hello ",
			},
		},
		{
			name: "string with default trimspace",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "stringval",
						String: &execution.StringOptionConfig{
							Default:    "hello ",
							TrimSpaces: true,
						},
					},
				},
			},
			want: map[string]string{
				"option.stringval": "hello",
			},
		},
		{
			name: "string without default required",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type:     execution.OptionTypeString,
						Name:     "stringval",
						Required: true,
						String:   &execution.StringOptionConfig{},
					},
				},
			},
			want: map[string]string{
				"option.stringval": "",
			},
		},
		{
			name: "string with default required",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type:     execution.OptionTypeString,
						Name:     "stringval",
						Required: true,
						String: &execution.StringOptionConfig{
							Default: "hello ",
						},
					},
				},
			},
			want: map[string]string{
				"option.stringval": "hello ",
			},
		},
		{
			name: "select without default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeSelect,
						Name: "selectval",
						Select: &execution.SelectOptionConfig{
							Values: []string{"a", "b"},
						},
					},
				},
			},
			want: map[string]string{
				"option.selectval": "",
			},
		},
		{
			name: "select with default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeSelect,
						Name: "selectval",
						Select: &execution.SelectOptionConfig{
							Values:  []string{"a", "b"},
							Default: "a",
						},
					},
				},
			},
			want: map[string]string{
				"option.selectval": "a",
			},
		},
		{
			name: "multi without default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeMulti,
						Name: "multival",
						Multi: &execution.MultiOptionConfig{
							Values:    []string{"a", "b"},
							Delimiter: ",",
						},
					},
				},
			},
			want: map[string]string{
				"option.multival": "",
			},
		},
		{
			name: "multi with default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeMulti,
						Name: "multival",
						Multi: &execution.MultiOptionConfig{
							Values:    []string{"a", "b"},
							Default:   []string{"a", "b"},
							Delimiter: ",",
						},
					},
				},
			},
			want: map[string]string{
				"option.multival": "a,b",
			},
		},
		{
			name: "date without default",
			cfg: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeDate,
						Name: "dateval",
						Date: &execution.DateOptionConfig{},
					},
				},
			},
			want: map[string]string{
				"option.dateval": "",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := options.MakeDefaultOptions(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeDefaultOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeDefaultOptions() got = %v, want %v", got, tt.want)
			}
		})
	}
}
