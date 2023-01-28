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

package completion_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/furiko-io/furiko/pkg/cli/completion"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
)

type testEnum string

func TestSliceCompleter(t *testing.T) {
	runtimetesting.RunCompleterTests(t, []runtimetesting.CompleterTest{
		{
			Name:      "nil will panic",
			Completer: completion.NewSliceCompleter(nil),
			WantError: assert.Error,
		},
		{
			Name:      "not a slice",
			Completer: completion.NewSliceCompleter("123"),
			WantError: assert.Error,
		},
		{
			Name:      "no items",
			Completer: completion.NewSliceCompleter([]string{}),
			Want:      []string{},
		},
		{
			Name:      "slice of string",
			Completer: completion.NewSliceCompleter([]string{"123", "456"}),
			Want:      []string{"123", "456"},
		},
		{
			Name:      "slice of int",
			Completer: completion.NewSliceCompleter([]int{123, 456}),
			Want:      []string{"123", "456"},
		},
		{
			Name:      "slice of enum",
			Completer: completion.NewSliceCompleter([]testEnum{"a", "b"}),
			Want:      []string{"a", "b"},
		},
	})
}
