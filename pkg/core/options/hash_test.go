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

func TestHashOptionsSpec(t *testing.T) {
	tests := []struct {
		name     string
		spec     *execution.OptionSpec
		equal    *execution.OptionSpec
		notEqual *execution.OptionSpec
		wantErr  bool
	}{
		{
			name:  "nil spec",
			equal: &execution.OptionSpec{},
		},
		{
			name: "empty spec",
			spec: &execution.OptionSpec{},
		},
		{
			name: "single option",
			spec: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "opt",
					},
				},
			},
		},
		{
			name: "empty pointer is equal",
			spec: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "opt",
					},
				},
			},
			equal: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type:   execution.OptionTypeString,
						Name:   "opt",
						String: &execution.StringOptionConfig{},
					},
				},
			},
		},
		{
			name: "reordered options are equal",
			spec: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "option1",
					},
					{
						Type: execution.OptionTypeString,
						Name: "option2",
					},
				},
			},
			equal: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "option2",
					},
					{
						Type: execution.OptionTypeString,
						Name: "option1",
					},
				},
			},
		},
		{
			name: "update config is not equal",
			spec: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "opt",
					},
				},
			},
			notEqual: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "opt",
						String: &execution.StringOptionConfig{
							TrimSpaces: true,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options.HashOptionSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashOptionSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.equal != nil {
				other, err := options.HashOptionSpec(tt.equal)
				if (err != nil) != tt.wantErr {
					t.Errorf("HashOptionSpec() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if other != got {
					t.Errorf("HashOptionSpec() returned different hashes: %v vs %v", got, other)
				}
			}
			if tt.notEqual != nil {
				other, err := options.HashOptionSpec(tt.notEqual)
				if (err != nil) != tt.wantErr {
					t.Errorf("HashOptionSpec() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if other == got {
					t.Errorf("HashOptionSpec() returned identical hashes: %v vs %v", got, other)
				}
			}
		})
	}
}

func TestHashOptionsSpec_Deterministic(t *testing.T) {
	tests := []struct {
		name string
		spec *execution.OptionSpec
	}{
		{
			name: "nil spec",
		},
		{
			name: "empty spec",
			spec: &execution.OptionSpec{},
		},
		{
			name: "single option",
			spec: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "option",
					},
				},
			},
		},
		{
			name: "multiple options",
			spec: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type: execution.OptionTypeString,
						Name: "string_option",
					},
					{
						Type:     execution.OptionTypeSelect,
						Name:     "select_option",
						Label:    "My Select Option",
						Required: true,
						Select: &execution.SelectOptionConfig{
							Default: "a",
							Values:  []string{"a", "b", "c"},
						},
					},
					{
						Type:     execution.OptionTypeMulti,
						Name:     "multi_option",
						Label:    "My Multi Option",
						Required: true,
						Multi: &execution.MultiOptionConfig{
							Default: []string{"b", "a"},
							Values:  []string{"a", "b", "c"},
						},
					},
					{
						Type:     execution.OptionTypeBool,
						Name:     "bool_option",
						Label:    "My Bool Option",
						Required: true,
						Bool: &execution.BoolOptionConfig{
							Default:  true,
							Format:   execution.BoolOptionFormatCustom,
							TrueVal:  "--true",
							FalseVal: "--false",
						},
					},
				},
			},
		},
	}

	const numIterations = 10
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hash string
			for i := 0; i < numIterations; i++ {
				got, err := options.HashOptionSpec(tt.spec)
				if err != nil {
					t.Errorf("HashOptionSpec() error = %v", err)
					return
				}
				if hash == "" {
					hash = got
				}
				if hash != got {
					t.Errorf("HashOptionSpec() returned a different hash: %v vs %v", hash, got)
					return
				}
			}
		})
	}
}
