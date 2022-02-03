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

package cmp

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
	"gomodules.xyz/jsonpatch/v2"
)

// IsJSONEqual returns whether the given objects are equal when marshaled to
// JSON. This performs a rather simple equality check, and may sometimes return
// false negatives if JSON keys are marshaled in a different order.
func IsJSONEqual(first, second interface{}) (bool, error) {
	firstBytes, secondBytes, err := marshalTwo(first, second)
	if err != nil {
		return false, err
	}
	return bytes.Equal(firstBytes, secondBytes), nil
}

// CreateJSONPatch returns a JSONPatch using the jsonpatch package.
func CreateJSONPatch(before, after interface{}) ([]byte, error) {
	rawBefore, rawAfter, err := marshalTwo(before, after)
	if err != nil {
		return nil, err
	}
	patch, err := jsonpatch.CreatePatch(rawBefore, rawAfter)
	if err != nil {
		return nil, err
	}
	return json.Marshal(patch)
}

func marshalTwo(first, second interface{}) ([]byte, []byte, error) {
	firstBytes, err := json.Marshal(first)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not json marshal %v", first)
	}
	secondBytes, err := json.Marshal(second)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not json marshal %v", second)
	}
	return firstBytes, secondBytes, nil
}
