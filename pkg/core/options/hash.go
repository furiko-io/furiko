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

package options

import (
	"reflect"
	"sort"
	"strconv"

	"github.com/mitchellh/hashstructure/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

var (
	defaultHashOptions = &hashstructure.HashOptions{
		TagName:         "json",
		ZeroNil:         true,
		IgnoreZeroValue: true,
	}
)

// HashOptionSpec returns a deterministic hash of the given OptionSpec.
// This can be used to easily determine if two Specs are equal or not.
func HashOptionSpec(spec *execution.OptionSpec) (string, error) {
	if spec == nil {
		spec = &execution.OptionSpec{}
	}

	newSpec := spec.DeepCopy()
	for i, option := range newSpec.Options {
		newOption := option.DeepCopy()

		// Drop fields that are not used during evaluation.
		newOption.Label = ""

		// Set all nil pointers to a zero-valued struct pointer.
		v := reflect.ValueOf(newOption).Elem()
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.Kind() == reflect.Ptr && f.IsZero() && f.CanAddr() {
				nilStruct := reflect.New(f.Type().Elem())
				f.Set(nilStruct)
			}
		}

		newSpec.Options[i] = *newOption
	}
	spec = newSpec

	// Sort the options.
	sort.Slice(spec.Options, func(i, j int) bool {
		return spec.Options[i].Name < spec.Options[j].Name
	})

	// Get the hash.
	hash, err := hashstructure.Hash(spec, hashstructure.FormatV2, defaultHashOptions)
	if err != nil {
		return "", err
	}

	// Use hex-encoded string.
	return strconv.FormatUint(hash, 16), nil
}
