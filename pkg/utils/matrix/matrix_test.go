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

package matrix_test

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/furiko-io/furiko/pkg/utils/matrix"
)

func TestGenerateMatrixCombinations(t *testing.T) {
	tests := []struct {
		name   string
		matrix matrix.Matrix
		want   []matrix.Combination
	}{
		{
			name: "empty matrix",
		},
		{
			name: "single dimension",
			matrix: map[string][]string{
				"key": {"a", "b", "c"},
			},
			want: []matrix.Combination{
				map[string]string{"key": "a"},
				map[string]string{"key": "b"},
				map[string]string{"key": "c"},
			},
		},
		{
			name: "single column",
			matrix: map[string][]string{
				"key": {"a"},
				"num": {"1"},
				"sym": {"!"},
			},
			want: []matrix.Combination{
				map[string]string{"key": "a", "num": "1", "sym": "!"},
			},
		},
		{
			name: "2x3 matrix",
			matrix: map[string][]string{
				"key": {"a", "b"},
				"num": {"1", "2", "3"},
			},
			want: []matrix.Combination{
				map[string]string{"key": "a", "num": "1"},
				map[string]string{"key": "a", "num": "2"},
				map[string]string{"key": "a", "num": "3"},
				map[string]string{"key": "b", "num": "1"},
				map[string]string{"key": "b", "num": "2"},
				map[string]string{"key": "b", "num": "3"},
			},
		},
		{
			name: "2x3x4 matrix",
			matrix: map[string][]string{
				"key": {"a", "b"},
				"num": {"1", "2", "3"},
				"sym": {"!", "@", "#", "$"},
			},
			want: []matrix.Combination{
				map[string]string{"key": "a", "num": "1", "sym": "!"},
				map[string]string{"key": "a", "num": "1", "sym": "@"},
				map[string]string{"key": "a", "num": "1", "sym": "#"},
				map[string]string{"key": "a", "num": "1", "sym": "$"},
				map[string]string{"key": "a", "num": "2", "sym": "!"},
				map[string]string{"key": "a", "num": "2", "sym": "@"},
				map[string]string{"key": "a", "num": "2", "sym": "#"},
				map[string]string{"key": "a", "num": "2", "sym": "$"},
				map[string]string{"key": "a", "num": "3", "sym": "!"},
				map[string]string{"key": "a", "num": "3", "sym": "@"},
				map[string]string{"key": "a", "num": "3", "sym": "#"},
				map[string]string{"key": "a", "num": "3", "sym": "$"},
				map[string]string{"key": "b", "num": "1", "sym": "!"},
				map[string]string{"key": "b", "num": "1", "sym": "@"},
				map[string]string{"key": "b", "num": "1", "sym": "#"},
				map[string]string{"key": "b", "num": "1", "sym": "$"},
				map[string]string{"key": "b", "num": "2", "sym": "!"},
				map[string]string{"key": "b", "num": "2", "sym": "@"},
				map[string]string{"key": "b", "num": "2", "sym": "#"},
				map[string]string{"key": "b", "num": "2", "sym": "$"},
				map[string]string{"key": "b", "num": "3", "sym": "!"},
				map[string]string{"key": "b", "num": "3", "sym": "@"},
				map[string]string{"key": "b", "num": "3", "sym": "#"},
				map[string]string{"key": "b", "num": "3", "sym": "$"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []cmp.Option{
				cmpopts.EquateEmpty(),
			}
			got := matrix.GenerateMatrixCombinations(tt.matrix)
			if !cmp.Equal(tt.want, got, opts...) {
				t.Errorf("GenerateMatrixCombinations() = not equal\ndiff = %v", cmp.Diff(tt.want, got, opts...))
			}
		})
	}
}

func TestGenerateMatrixCombinations_Deterministic(t *testing.T) {
	input := matrix.Matrix{
		"key": {"a", "b"},
		"num": {"1", "2", "3"},
		"sym": {"!", "@", "#", "$"},
	}
	prev := matrix.GenerateMatrixCombinations(input)
	for i := 0; i < 10000; i++ {
		output := matrix.GenerateMatrixCombinations(input)
		if !reflect.DeepEqual(prev, output) {
			t.Errorf("not deterministic")
			return
		}
		prev = output
	}
}
