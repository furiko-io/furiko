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

package strings_test

import (
	"testing"

	"github.com/furiko-io/furiko/pkg/utils/strings"
)

func TestContainsString(t *testing.T) {
	tests := []struct {
		name string
		strs []string
		s    string
		want bool
	}{
		{
			name: "empty list",
			strs: nil,
			s:    "search",
			want: false,
		},
		{
			name: "exact match",
			strs: []string{"search"},
			s:    "search",
			want: true,
		},
		{
			name: "skip non-matches",
			strs: []string{"", "abc", "search"},
			s:    "search",
			want: true,
		},
		{
			name: "not substring match",
			strs: []string{"", "abc", "can_you_search_for_me"},
			s:    "search",
			want: false,
		},
		{
			name: "case sensitive match",
			strs: []string{"", "abc", "Search"},
			s:    "search",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strings.ContainsString(tt.strs, tt.s); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}
