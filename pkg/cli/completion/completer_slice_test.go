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

package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

type testEnum string

func Test_sliceToStrings(t *testing.T) {
	tests := []struct {
		name    string
		slice   any
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "nil will panic",
			wantErr: assert.Error,
		},
		{
			name:    "not a slice",
			slice:   "123",
			wantErr: assert.Error,
		},
		{
			name:  "slice of string",
			slice: []string{"123", "456"},
			want:  []string{"123", "456"},
		},
		{
			name:  "slice of int",
			slice: []int{123, 456},
			want:  []string{"123", "456"},
		},
		{
			name:  "slice of enum",
			slice: []testEnum{"a", "b"},
			want:  []string{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sliceToStrings(tt.slice)
			if testutils.WantError(t, tt.wantErr, err) {
				return
			}
			assert.Equalf(t, tt.want, got, "sliceToStrings(%v) not equal", tt.slice)
		})
	}
}
