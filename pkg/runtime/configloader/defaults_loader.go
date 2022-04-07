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

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/furiko-io/furiko/apis/config/v1"
	"github.com/furiko-io/furiko/pkg/config"
)

// DefaultsLoader is a simple ConfigLoader that populates hardcoded defaults.
// This is intended to be a fallback when all configs are not populated.
type DefaultsLoader struct {
	cache sync.Map

	// Can override defaults here, but they are not expected to reload after Start()
	// is called. Exposed for unit tests.
	Defaults map[v1.ConfigName]runtime.Object
}

func NewDefaultsLoader() *DefaultsLoader {
	return &DefaultsLoader{
		Defaults: map[v1.ConfigName]runtime.Object{
			v1.ConfigNameJobController:  config.DefaultJobControllerConfig,
			v1.ConfigNameCronController: config.DefaultCronControllerConfig,
		},
	}
}

func (d *DefaultsLoader) Name() string {
	return "DefaultsLoader"
}

func (d *DefaultsLoader) Start(_ context.Context) error {
	for configName, object := range d.Defaults {
		cfg, err := d.marshalToViper(object)
		if err != nil {
			return err
		}
		d.cache.Store(configName, cfg)
	}
	return nil
}

func (d *DefaultsLoader) GetConfig(configName v1.ConfigName) (*viper.Viper, error) {
	value, ok := d.cache.Load(configName)
	if !ok {
		return viper.New(), nil
	}
	loaded, ok := value.(*viper.Viper)
	if !ok {
		return nil, fmt.Errorf("unexpected type, got %T", value)
	}
	return loaded, nil
}

func (d *DefaultsLoader) marshalToViper(in interface{}) (*viper.Viper, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	v := viper.New()
	if err := v.MergeConfigMap(out); err != nil {
		return nil, err
	}
	return v, nil
}
