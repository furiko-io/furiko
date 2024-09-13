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

	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
)

func TestZeroForNonConfig(t *testing.T) {
	tests := []struct {
		name   string
		option v1alpha1.Option
		want   v1alpha1.Option
	}{
		{
			name: "empty value",
		},
		{
			name: "only non-config fields set",
			option: v1alpha1.Option{
				Type:     v1alpha1.OptionTypeString,
				Name:     "myOption",
				Label:    "My Option",
				Required: true,
			},
		},
		{
			name: "some config fields set",
			option: v1alpha1.Option{
				Type:     v1alpha1.OptionTypeString,
				Name:     "myOption",
				Label:    "My Option",
				Required: true,
				Bool:     &v1alpha1.BoolOptionConfig{},
				String: &v1alpha1.StringOptionConfig{
					Default:    "defaultValue",
					TrimSpaces: true,
				},
			},
			want: v1alpha1.Option{
				Bool: &v1alpha1.BoolOptionConfig{},
				String: &v1alpha1.StringOptionConfig{
					Default:    "defaultValue",
					TrimSpaces: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := options.ZeroForNonConfig(tt.option); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ZeroForNonConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
