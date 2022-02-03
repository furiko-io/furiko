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
	"reflect"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
)

// ConfigManager manages ConfigLoaders and merges structured configuration
// values from multiple sources. The order in which the configurations are
// merged are based on the order of when each ConfigLoader is added to the
// ConfigManager.
type ConfigManager struct {
	loaders []ConfigLoader
	started bool
	cache   sync.Map
}

func NewConfigManager() *ConfigManager {
	return &ConfigManager{}
}

func (c *ConfigManager) AddConfigLoaders(loader ...ConfigLoader) {
	c.loaders = append(c.loaders, loader...)
}

func (c *ConfigManager) Start(ctx context.Context) error {
	for _, loader := range c.loaders {
		if err := loader.Start(ctx); err != nil {
			return errors.Wrapf(err, "cannot load %v", loader.Name())
		}
	}
	c.started = true
	return nil
}

// LoadAndUnmarshalConfig will load and unmarshal the given config name into out.
//
// If an error is encountered, it will return a previously known good value if
// available, and log the error. Otherwise, if there is no previously cached
// value for configName, then the error will be propagated back to the caller.
func (c *ConfigManager) LoadAndUnmarshalConfig(configName configv1.ConfigName, out interface{}) error {
	err := c.loadAndUnmarshalConfigWithError(configName, out)

	// Return cached value and log error.
	// We use reflection to write into the value referenced by the pointer out.
	if err != nil {
		// Sanity check here, out should be an addressable pointer.
		outVal := reflect.ValueOf(out)
		if outVal.Kind() != reflect.Ptr {
			return errors.New("out must be a pointer")
		}
		outVal = outVal.Elem()
		if !outVal.CanAddr() {
			return errors.New("out must be addressable (a pointer)")
		}

		// Here we load the previously cached value into the pointer.
		loadVal, ok := c.cache.Load(configName)
		if ok {
			dataVal := reflect.ValueOf(loadVal)

			// Take the indirect reference of loadVal if it is a pointer (it should be).
			if dataVal.Kind() == reflect.Ptr && dataVal.Type().Elem() == outVal.Type() {
				dataVal = reflect.Indirect(dataVal)
			}

			outVal.Set(dataVal)

			klog.ErrorS(err, "configloader: load config failed, falling back to last known good value",
				"configName", configName)
			return nil
		}

		// Forward error if there is no cached value.
		klog.ErrorS(err, "configloader: load config failed, no previously known good value",
			"configName", configName)
		return err
	}

	// Store in cache.
	c.cache.Store(configName, out)
	return nil
}

func (c *ConfigManager) loadAndUnmarshalConfigWithError(configName configv1.ConfigName, out interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  out,
	})
	if err != nil {
		return err
	}
	configMap, err := c.loadConfig(configName)
	if err != nil {
		return errors.Wrapf(err, "cannot load config %v", configName)
	}
	if err := decoder.Decode(configMap.AllSettings()); err != nil {
		return errors.Wrapf(err, "cannot decode %v", configName)
	}
	return nil
}

// loadConfig will load the given config name from all loaders.
func (c *ConfigManager) loadConfig(configName configv1.ConfigName) (*viper.Viper, error) {
	if !c.started {
		return nil, errors.New("config manager is not started")
	}
	final := viper.New()
	for _, loader := range c.loaders {
		loaded, err := loader.GetConfig(configName)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot load %v", loader.Name())
		}
		if err := final.MergeConfigMap(loaded.AllSettings()); err != nil {
			return nil, errors.Wrapf(err, "cannot merge configs")
		}
	}
	return final, nil
}
