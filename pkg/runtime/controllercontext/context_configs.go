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

package controllercontext

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/configloader"
)

// ConfigsMap is a map of ConfigName to Config object.
type ConfigsMap = map[configv1alpha1.ConfigName]runtime.Object

// Configs returns the dynamic controller configurations.
func (c *ctrlContext) Configs() Configs {
	return c.configMgr
}

type Configs interface {
	Start(ctx context.Context) error
	AllConfigs() (map[configv1alpha1.ConfigName]runtime.Object, error)
	Jobs() (*configv1alpha1.JobExecutionConfig, error)
	Cron() (*configv1alpha1.CronExecutionConfig, error)
}

type ContextConfigs struct {
	*configloader.ConfigManager
}

func NewContextConfigs(mgr *configloader.ConfigManager) *ContextConfigs {
	return &ContextConfigs{ConfigManager: mgr}
}

// AllConfigs returns a map of all configs.
func (c *ContextConfigs) AllConfigs() (map[configv1alpha1.ConfigName]runtime.Object, error) {
	configNameMap := map[configv1alpha1.ConfigName]func() (runtime.Object, error){
		configv1alpha1.JobExecutionConfigName: func() (runtime.Object, error) {
			return c.Jobs()
		},
		configv1alpha1.CronExecutionConfigName: func() (runtime.Object, error) {
			return c.Cron()
		},
	}

	configs := make(map[configv1alpha1.ConfigName]runtime.Object)
	for configName, load := range configNameMap {
		cfg, err := load()
		if err != nil {
			return nil, errors.Wrapf(err, "cannot load config for %v", configName)
		}
		configs[configName] = cfg
	}

	return configs, nil
}

// Jobs returns the job dynamic configuration.
func (c *ContextConfigs) Jobs() (*configv1alpha1.JobExecutionConfig, error) {
	var config configv1alpha1.JobExecutionConfig
	if err := c.LoadAndUnmarshalConfig(configv1alpha1.JobExecutionConfigName, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// Cron returns the cron dynamic configuration.
func (c *ContextConfigs) Cron() (*configv1alpha1.CronExecutionConfig, error) {
	var config configv1alpha1.CronExecutionConfig
	if err := c.LoadAndUnmarshalConfig(configv1alpha1.CronExecutionConfigName, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SetUpConfigManager sets up the ConfigManager and returns a composed Configs interface.
func SetUpConfigManager(cfg *configv1alpha1.BootstrapConfigSpec, client kubernetes.Interface) Configs {
	configManager := configloader.NewConfigManager()
	var configMapNamespace, configMapName, secretNamespace, secretName string
	if cfg := cfg.DynamicConfigs; cfg != nil {
		if cfg := cfg.ConfigMap; cfg != nil {
			configMapNamespace = cfg.Namespace
			configMapName = cfg.Name
		}
		if cfg := cfg.Secret; cfg != nil {
			secretNamespace = cfg.Namespace
			secretName = cfg.Name
		}
	}
	configManager.AddConfigLoaders(
		configloader.NewDefaultsLoader(),
		configloader.NewConfigMapLoader(client, configMapNamespace, configMapName),
		configloader.NewSecretLoader(client, secretNamespace, secretName),
	)
	return NewContextConfigs(configManager)
}
