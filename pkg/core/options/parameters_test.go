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

	"github.com/furiko-io/furiko/pkg/core/options"
)

func TestFilterParametersWithPrefix(t *testing.T) {
	type args struct {
		parameters map[string]string
		prefix     string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "nil map",
			args: args{
				parameters: nil,
				prefix:     "option.",
			},
			want: map[string]string{},
		},
		{
			name: "empty map",
			args: args{
				parameters: map[string]string{},
				prefix:     "option.",
			},
			want: map[string]string{},
		},
		{
			name: "filter in prefix",
			args: args{
				parameters: map[string]string{
					"option.abc":   "abc",
					"pod.nodename": "node-1",
				},
				prefix: "option.",
			},
			want: map[string]string{
				"option.abc": "abc",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := options.FilterParametersWithPrefix(tt.args.parameters, tt.args.prefix); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterParametersWithPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterParametersWithPrefixes(t *testing.T) {
	type args struct {
		parameters map[string]string
		prefixes   []string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "nil map",
			args: args{
				parameters: nil,
				prefixes:   []string{"option."},
			},
			want: map[string]string{},
		},
		{
			name: "empty map",
			args: args{
				parameters: map[string]string{},
				prefixes:   []string{"option."},
			},
			want: map[string]string{},
		},
		{
			name: "filter in prefix",
			args: args{
				parameters: map[string]string{
					"option.abc":   "abc",
					"pod.nodename": "node-1",
					"task.id":      "123",
				},
				prefixes: []string{"option."},
			},
			want: map[string]string{
				"option.abc": "abc",
			},
		},
		{
			name: "filter in multiple prefixes",
			args: args{
				parameters: map[string]string{
					"option.abc":   "abc",
					"pod.nodename": "node-1",
					"task.id":      "123",
				},
				prefixes: []string{"job.", "task.", "option."},
			},
			want: map[string]string{
				"option.abc": "abc",
				"task.id":    "123",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := options.FilterParametersWithPrefixes(tt.args.parameters, tt.args.prefixes...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterParametersWithPrefixes() = %v, want %v", got, tt.want)
			}
		})
	}
}
