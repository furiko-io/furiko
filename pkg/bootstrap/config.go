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

package bootstrap

import (
	"os"

	"github.com/pkg/errors"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

// UnmarshalFromFile reads and unmarshals a YAML or JSON file into out.
// If filePath is empty, nothing will be done.
func UnmarshalFromFile(filePath string, out interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return errors.Wrapf(err, "cannot open %v", filePath)
	}
	defer func() { _ = file.Close() }()
	decoder := utilyaml.NewYAMLOrJSONDecoder(file, 4096)
	return decoder.Decode(out)
}

// Marshal marshals the given input into a pretty-printed YAML string.
func Marshal(input interface{}) (string, error) {
	data, err := yaml.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
