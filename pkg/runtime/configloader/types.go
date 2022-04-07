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

package configloader

import (
	"context"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
)

// Config is a dynamic config value.
type Config map[string]interface{}

// Loader knows how to loads a Config given a config name.
type Loader interface {
	Name() string
	Start(context.Context) error
	Load(configName configv1.ConfigName) (Config, error)
}
