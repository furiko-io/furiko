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

package jsonyaml

import (
	"bytes"

	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	bufferSize = 4096
)

// Unmarshal parses the JSON or YAML-encoded data and stores the result in the
// value pointed to by v.
func Unmarshal(data []byte, v interface{}) error {
	return yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), bufferSize).Decode(v)
}

// UnmarshalString parses the JSON or YAML-encoded string and stores the result
// in the value pointed to by v.
func UnmarshalString(data string, v interface{}) error {
	return yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(data), bufferSize).Decode(v)
}
