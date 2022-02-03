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

package mock

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	"github.com/furiko-io/furiko/pkg/configloader"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

type Configs struct {
	controllercontext.Configs
	configLoader *ConfigLoader
}

// NewConfigs returns a new dynamic Configs manager that supports overriding a
// defaults configloader with configs from memory.
func NewConfigs() *Configs {
	mgr := configloader.NewConfigManager()
	configLoader := NewMockConfigLoader()
	mgr.AddConfigLoaders(
		configloader.NewDefaultsLoader(),
		configLoader,
	)
	return &Configs{
		Configs:      controllercontext.NewContextConfigs(mgr),
		configLoader: configLoader,
	}
}

// MockConfigLoader returns the mock configloader.ConfigLoader.
func (c *Configs) MockConfigLoader() *ConfigLoader {
	return c.configLoader
}

var _ controllercontext.Configs = &Configs{}

type ConfigLoader struct {
	configs map[configv1.ConfigName]runtime.Object
	mu      sync.RWMutex
}

func NewMockConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		configs: make(map[configv1.ConfigName]runtime.Object),
	}
}

func (c *ConfigLoader) Name() string {
	return "Mock"
}

func (c *ConfigLoader) Start(ctx context.Context) error {
	return nil
}

func (c *ConfigLoader) GetConfig(configName configv1.ConfigName) (*viper.Viper, error) {
	v := viper.New()
	config, ok := c.configs[configName]
	if !ok {
		return v, nil
	}
	bytes, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &m); err != nil {
		return nil, err
	}
	if err := v.MergeConfigMap(m); err != nil {
		return nil, err
	}
	return v, nil
}

func (c *ConfigLoader) SetConfig(configName configv1.ConfigName, config runtime.Object) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configs[configName] = config
}
