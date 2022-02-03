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
	"encoding/base64"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	v1 "github.com/furiko-io/furiko/apis/config/v1"
	"github.com/furiko-io/furiko/pkg/utils/eventhandler"
)

const (
	defaultScrtNamespace = "furiko-system"
	defaultScrtName      = "execution-dynamic-config"
)

// SecretLoader is a dynamic ConfigLoader that starts an informer to watch
// changes on a Secret with a specific name and namespace. Supports loading
// both JSON and YAML configuration.
type SecretLoader struct {
	*ConfigMapLoader
}

func NewSecretLoader(client kubernetes.Interface, namespace, name string) *SecretLoader {
	if namespace == "" {
		namespace = defaultScrtNamespace
	}
	if name == "" {
		name = defaultScrtName
	}
	return &SecretLoader{
		ConfigMapLoader: NewConfigMapLoader(client, namespace, name),
	}
}

func (c *SecretLoader) Name() string {
	return "SecretLoader"
}

func (c *SecretLoader) Start(ctx context.Context) error {
	// Create shared informer factory watching only the specified namespace for Secrets.
	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.client, time.Minute*10,
		informers.WithNamespace(c.namespace))
	informer := informerFactory.Core().V1().Secrets().Informer()
	return c.startInformer(ctx, informerFactory, informer, c.handleUpdate)
}

func (c *SecretLoader) handleUpdate(obj interface{}) {
	secret, err := eventhandler.Corev1Secret(obj)
	if err != nil {
		klog.ErrorS(err, "configloader: unable to handle event", "loader", c.Name())
		return
	}

	// Ignore update if it is not the Secret we are watching.
	if secret.Name != c.name || secret.Namespace != c.namespace {
		return
	}

	// Don't log data to avoid leaking secrets.
	klog.V(4).InfoS("configloader: config loader observed update",
		"loader", c.Name(),
		"name", secret.Name,
		"namespace", secret.Namespace,
	)

	newSecret, err := c.unmarshalSecret(secret.Data)
	if err != nil {
		klog.ErrorS(err, "configloader: config unmarshal error", "loader", c.Name())
		return
	}
	c.cache = newSecret
}

func (c *SecretLoader) unmarshalSecret(data map[string][]byte) (*configCache, error) {
	newSecret := newConfigCache()
	for name, value := range data {
		bytes, err := base64.StdEncoding.DecodeString(string(value))
		if err != nil {
			return nil, errors.Wrapf(err, "cannot decode base64 data")
		}
		v, err := c.unmarshalToViper(string(bytes))
		if err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal %v", name)
		}
		newSecret.Store(v1.ConfigName(name), v)
	}
	return newSecret, nil
}
