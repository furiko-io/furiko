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
	"context"
	"fmt"
	"reflect"

	"github.com/spf13/cobra"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

type sliceCompleter struct {
	items any
}

var _ Completer = (*sliceCompleter)(nil)

// NewSliceCompleter is a Completer from a slice of objects.
func NewSliceCompleter(items any) Completer {
	return &sliceCompleter{
		items: items,
	}
}

func (c *sliceCompleter) Complete(_ context.Context, _ controllercontext.Context, _ *cobra.Command) ([]string, error) {
	return sliceToStrings(c.items)
}

func sliceToStrings(slice any) ([]string, error) {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input is not a slice: %v", slice)
	}
	items := make([]string, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		items = append(items, fmt.Sprintf("%v", v.Index(i)))
	}
	return items, nil
}
