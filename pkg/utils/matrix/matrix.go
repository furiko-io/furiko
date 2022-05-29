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

package matrix

import (
	"sort"
)

// Matrix is a map of keys to list of possible values.
type Matrix map[string][]string

// Combination is a single combination of matrix values.
type Combination map[string]string

// GenerateMatrixCombinations returns a cartesian product of all possible
// key-value combinations in matrix, and returns a list of all map values.
func GenerateMatrixCombinations(matrix Matrix) []Combination {
	totalCombinations := NumCombinations(matrix)
	keys := GetKeys(matrix)
	indexes := make([]int, len(keys))
	combinations := make([]Combination, 0, totalCombinations)
	for i := 0; i < totalCombinations; i++ {
		// Fix indexes.
		for j := len(indexes) - 1; j >= 0; j-- {
			if indexes[j] >= len(matrix[keys[j]]) {
				indexes[j] = 0
				indexes[j-1]++
			}
		}

		// Generate new combination.
		combinations = append(combinations, IndexMatrix(matrix, keys, indexes))

		// Advance indexes.
		indexes[len(indexes)-1]++
	}

	return combinations
}

// IndexMatrix indexes a matrix on indexes with keys and returns a Combination.
func IndexMatrix(matrix Matrix, keys []string, indexes []int) Combination {
	combi := make(Combination)
	for k, idx := range indexes {
		key := keys[k]
		combi[key] = matrix[key][idx]
	}
	return combi
}

// GetKeys returns a sorted list of keys in a Matrix.
func GetKeys(matrix Matrix) []string {
	keys := make([]string, 0, len(matrix))
	for key := range matrix {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// NumCombinations returns the total number of combinations for a Matrix.
func NumCombinations(matrix Matrix) int {
	var total int
	for _, values := range matrix {
		if total == 0 {
			total = 1
		}
		total *= len(values)
	}
	return total
}
