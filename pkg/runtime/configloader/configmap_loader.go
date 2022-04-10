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
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/eventhandler"
)

const (
	defaultConfigMapNamespace = "furiko-system"
	defaultConfigMapName      = "execution-dynamic-config"
)

// ConfigMapLoader is a dynamic Loader that starts an informer to watch
// changes on a ConfigMap with a specific name and namespace. Supports loading
// both JSON and YAML configuration.
type ConfigMapLoader struct {
	client    kubernetes.Interface
	cache     *configCache
	namespace string
	name      string
}

var _ Loader = (*ConfigMapLoader)(nil)

func NewConfigMapLoader(client kubernetes.Interface, namespace, name string) *ConfigMapLoader {
	if namespace == "" {
		namespace = defaultConfigMapNamespace
	}
	if name == "" {
		name = defaultConfigMapName
	}
	return &ConfigMapLoader{
		client:    client,
		cache:     newConfigCache(),
		namespace: namespace,
		name:      name,
	}
}

func (c *ConfigMapLoader) Name() string {
	return "ConfigMapLoader"
}

func (c *ConfigMapLoader) Start(ctx context.Context) error {
	// Create shared informer factory watching only the specified namespace for ConfigMaps.
	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.client, time.Minute*10,
		informers.WithNamespace(c.namespace))
	informer := informerFactory.Core().V1().ConfigMaps().Informer()
	return c.startInformer(ctx, informerFactory, informer, c.handleUpdate)
}

func (c *ConfigMapLoader) startInformer(
	ctx context.Context, informerFactory informers.SharedInformerFactory, informer cache.SharedIndexInformer,
	eventHandler func(obj interface{}),
) error {
	klog.V(4).InfoS("configloader: config loader starting", "loader", c.Name())
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: eventHandler,
		UpdateFunc: func(oldObj, newObj interface{}) {
			eventHandler(newObj)
		},
	})

	informerFactory.Start(ctx.Done())

	// Wait for caches to be synced with a timeout.
	syncCtx, cancel := context.WithTimeout(ctx, time.Minute*3)
	defer cancel()
	if ok := cache.WaitForNamedCacheSync(c.Name(), syncCtx.Done(), informer.HasSynced); !ok {
		klog.Error("configloader: failed to sync caches", "loader", c.Name())
		return errors.New("failed to sync caches")
	}

	return nil
}

// Load returns the unmarshaled config data stored in a given ConfigMap. If the
// ConfigMap or config name in the ConfigMap does not exist, an empty config
// will be returned.
func (c *ConfigMapLoader) Load(configName configv1alpha1.ConfigName) (Config, error) {
	if value, ok := c.cache.Load(configName); ok {
		return value, nil
	}
	return nil, nil
}

func (c *ConfigMapLoader) handleUpdate(obj interface{}) {
	cm, err := eventhandler.Corev1ConfigMap(obj)
	if err != nil {
		klog.ErrorS(err, "configloader: unable to handle event", "loader", c.Name())
		return
	}

	// Ignore update if it is not the ConfigMap we are watching.
	if cm.Name != c.name || cm.Namespace != c.namespace {
		return
	}

	klog.V(4).InfoS("configloader: config loader observed update",
		"loader", c.Name(),
		"name", cm.Name,
		"namespace", cm.Namespace,
		"data", spew.Sdump(cm.Data),
	)

	newConfigMap, err := c.unmarshalConfigMap(cm.Data)
	if err != nil {
		klog.ErrorS(err, "configloader: config unmarshal error", "loader", c.Name())
		return
	}
	c.cache = newConfigMap
}

func (c *ConfigMapLoader) unmarshalConfigMap(data map[string]string) (*configCache, error) {
	newConfigMap := newConfigCache()
	for name, value := range data {
		v, err := c.unmarshal(value)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal %v: %v", name, value)
		}
		newConfigMap.Store(configv1alpha1.ConfigName(name), v)
	}
	return newConfigMap, nil
}

func (c *ConfigMapLoader) unmarshal(data string) (Config, error) {
	var conf Config
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(data), 4096)
	if err := decoder.Decode(&conf); err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshal configmap data: %v", data)
	}
	return conf, nil
}

type configCache struct {
	m *sync.Map
}

func newConfigCache() *configCache {
	return &configCache{
		m: &sync.Map{},
	}
}

func (c *configCache) Store(configName configv1alpha1.ConfigName, config Config) {
	c.m.Store(configName, config)
}

func (c *configCache) Load(configName configv1alpha1.ConfigName) (Config, bool) {
	v, ok := c.m.Load(configName)
	if !ok {
		return nil, false
	}
	return v.(Config), true
}
