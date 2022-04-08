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
	"encoding/json"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
)

// DefaultsLoader is a simple Loader that populates hardcoded defaults.
// This is intended to be a fallback when all configs are not populated.
type DefaultsLoader struct {
	cache sync.Map

	// Can override defaults here, but they are not expected to reload after Start()
	// is called. Exposed for unit tests.
	Defaults map[v1.ConfigName]runtime.Object
}

var _ Loader = (*DefaultsLoader)(nil)

func NewDefaultsLoader() *DefaultsLoader {
	return &DefaultsLoader{
		Defaults: map[v1.ConfigName]runtime.Object{
			v1.JobExecutionConfigName:  config.DefaultJobControllerConfig,
			v1.CronExecutionConfigName: config.DefaultCronControllerConfig,
		},
	}
}

func (d *DefaultsLoader) Name() string {
	return "DefaultsLoader"
}

func (d *DefaultsLoader) Start(_ context.Context) error {
	for configName, object := range d.Defaults {
		cfg, err := d.marshal(object)
		if err != nil {
			return err
		}
		d.cache.Store(configName, cfg)
	}
	return nil
}

func (d *DefaultsLoader) Load(configName v1.ConfigName) (Config, error) {
	value, ok := d.cache.Load(configName)
	if !ok {
		return nil, nil
	}
	loaded, ok := value.(Config)
	if !ok {
		return nil, fmt.Errorf("unexpected type, got %T", value)
	}
	return loaded, nil
}

func (d *DefaultsLoader) marshal(in interface{}) (Config, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out Config
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
