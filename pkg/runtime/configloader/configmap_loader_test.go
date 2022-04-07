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

package configloader_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	"github.com/furiko-io/furiko/pkg/runtime/configloader"
)

const (
	fakeclientsetSleepDuration = time.Millisecond * 10

	configMapNamespace = "test-namespace"
	configMapName      = "test-config"
)

type ConfigLoaderControl interface {
	Create(ctx context.Context, client kubernetes.Interface) error
	Update(ctx context.Context, client kubernetes.Interface, data string) error
}

type configMapLoaderTest struct{}

func (c *configMapLoaderTest) Create(ctx context.Context, client kubernetes.Interface) error {
	_, err := client.CoreV1().ConfigMaps(configMapNamespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: configMapNamespace,
			Name:      configMapName,
		},
	}, metav1.CreateOptions{})
	return err
}

func (c *configMapLoaderTest) Update(ctx context.Context, client kubernetes.Interface, data string) error {
	_, err := client.CoreV1().ConfigMaps(configMapNamespace).Update(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: configMapNamespace,
			Name:      configMapName,
		},
		Data: map[string]string{
			string(configv1.ConfigNameJobController): data,
		},
	}, metav1.UpdateOptions{})
	return err
}

func TestConfigMapLoader(t *testing.T) {
	client := fakeclientset.NewSimpleClientset()
	loader := configloader.NewConfigMapLoader(client, configMapNamespace, configMapName)
	testKubernetesLoader(t, client, loader, &configMapLoaderTest{})
}

func testKubernetesLoader(
	t *testing.T, client kubernetes.Interface, loader configloader.Loader, ctrl ConfigLoaderControl,
) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	create := func() {
		err := ctrl.Create(ctx, client)
		assert.NoError(t, err)
		time.Sleep(fakeclientsetSleepDuration)
	}

	update := func(data string) {
		err := ctrl.Update(ctx, client, data)
		assert.NoError(t, err)
		time.Sleep(fakeclientsetSleepDuration)
	}

	mgr := configloader.NewConfigManager()
	mgr.AddConfigLoaders(loader)
	err := mgr.Start(ctx)
	assert.NoError(t, err)

	// Load empty config, should have no values
	cfg, err := loadJobControllerConfig(mgr)
	assert.NoError(t, err)
	assert.Zero(t, cfg.DefaultPendingTimeoutSeconds)
	assert.Zero(t, cfg.DefaultTTLSecondsAfterFinished)

	// Create with no data initially
	create()
	cfg, err = loadJobControllerConfig(mgr)
	assert.NoError(t, err)
	assert.Zero(t, cfg.DefaultPendingTimeoutSeconds)
	assert.Zero(t, cfg.DefaultTTLSecondsAfterFinished)

	// Store some JSON
	update(`{"defaultPendingTimeoutSeconds": 180}`)
	cfg, err = loadJobControllerConfig(mgr)
	assert.NoError(t, err)
	assert.Equal(t, pointer.Int64(180), cfg.DefaultPendingTimeoutSeconds)
	assert.Zero(t, cfg.DefaultTTLSecondsAfterFinished)

	// Update value
	update(`{"defaultPendingTimeoutSeconds": 180, "defaultTTLSecondsAfterFinished": 3600}`)
	cfg, err = loadJobControllerConfig(mgr)
	assert.NoError(t, err)
	assert.Equal(t, pointer.Int64(180), cfg.DefaultPendingTimeoutSeconds)
	assert.Equal(t, pointer.Int64(3600), cfg.DefaultTTLSecondsAfterFinished)

	// Store invalid JSON, previous values should still be retained
	update(`{"defaultPendingTimeoutSeconds": 190, "defaultTTLSecondsAfterFinished": 3601`)
	cfg, err = loadJobControllerConfig(mgr)
	assert.NoError(t, err)
	assert.Equal(t, pointer.Int64(180), cfg.DefaultPendingTimeoutSeconds)
	assert.Equal(t, pointer.Int64(3600), cfg.DefaultTTLSecondsAfterFinished)

	// Fix the JSON, now values should be updated
	update(`{"defaultPendingTimeoutSeconds": 190, "defaultTTLSecondsAfterFinished": 3601}`)
	cfg, err = loadJobControllerConfig(mgr)
	assert.NoError(t, err)
	assert.Equal(t, pointer.Int64(190), cfg.DefaultPendingTimeoutSeconds)
	assert.Equal(t, pointer.Int64(3601), cfg.DefaultTTLSecondsAfterFinished)

	// Store YAML instead
	update(`---
defaultTTLSecondsAfterFinished: 1234
defaultPendingTimeoutSeconds: 2345
`)
	cfg, err = loadJobControllerConfig(mgr)
	assert.NoError(t, err)
	assert.Equal(t, pointer.Int64(1234), cfg.DefaultTTLSecondsAfterFinished)
	assert.Equal(t, pointer.Int64(2345), cfg.DefaultPendingTimeoutSeconds)

	// Store invalid YAML, previous values should still be retained
	update(`---
defaultTTLSecondsAfterFinished: 512
	defaultPendingTimeoutSeconds: 12312
`)
	cfg, err = loadJobControllerConfig(mgr)
	assert.NoError(t, err)
	assert.Equal(t, pointer.Int64(1234), cfg.DefaultTTLSecondsAfterFinished)
	assert.Equal(t, pointer.Int64(2345), cfg.DefaultPendingTimeoutSeconds)

	// Fix the YAML, now should be updated
	update(`---
defaultTTLSecondsAfterFinished: 512
defaultPendingTimeoutSeconds: 12312
`)
	cfg, err = loadJobControllerConfig(mgr)
	assert.NoError(t, err)
	assert.Equal(t, pointer.Int64(512), cfg.DefaultTTLSecondsAfterFinished)
	assert.Equal(t, pointer.Int64(12312), cfg.DefaultPendingTimeoutSeconds)
}
