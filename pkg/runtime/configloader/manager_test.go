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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	"github.com/furiko-io/furiko/pkg/runtime/configloader"
)

type MockConfig map[configv1.ConfigName]map[string]interface{}

type mockLoader struct {
}

func (m *mockLoader) Name() string {
	return "Mock"
}

func (m *mockLoader) Start(_ context.Context) error {
	return nil
}

type mockErrorLoader struct {
	*mockLoader
}

func newMockErrorLoader() *mockErrorLoader {
	return &mockErrorLoader{
		mockLoader: &mockLoader{},
	}
}

func (m *mockErrorLoader) GetConfig(_ configv1.ConfigName) (*viper.Viper, error) {
	return nil, errors.New("get config error")
}

type mockConfigLoader struct {
	*mockLoader
	values MockConfig
}

func newMockConfigLoader(values MockConfig) *mockConfigLoader {
	return &mockConfigLoader{
		mockLoader: &mockLoader{},
		values:     values,
	}
}

func (m *mockConfigLoader) GetConfig(configName configv1.ConfigName) (*viper.Viper, error) {
	v := viper.New()
	if err := v.MergeConfigMap(m.values[configName]); err != nil {
		return nil, err
	}
	return v, nil
}

type mockDynamicConfigLoader struct {
	*mockConfigLoader
}

func newMockDynamicConfigLoader(values MockConfig) *mockDynamicConfigLoader {
	return &mockDynamicConfigLoader{
		mockConfigLoader: newMockConfigLoader(values),
	}
}

func (m *mockDynamicConfigLoader) SetConfig(config MockConfig) {
	m.values = config
}

func TestConfigManager(t *testing.T) {
	tests := []struct {
		name    string
		loaders []configloader.ConfigLoader
		want    *configv1.JobControllerConfig
		wantErr bool
	}{
		{
			name:    "no loaders",
			loaders: nil,
			want:    &configv1.JobControllerConfig{},
		},
		{
			name: "loader error",
			loaders: []configloader.ConfigLoader{
				newMockErrorLoader(),
			},
			wantErr: true,
		},
		{
			name: "single loader",
			loaders: []configloader.ConfigLoader{
				newMockConfigLoader(MockConfig{
					configv1.ConfigNameJobController: {
						"defaultTTLSecondsAfterFinished": 180,
						"defaultPendingTimeoutSeconds":   900,
					},
				}),
			},
			want: &configv1.JobControllerConfig{
				DefaultTTLSecondsAfterFinished: 180,
				DefaultPendingTimeoutSeconds:   900,
			},
		},
		{
			name: "override values and add new fields in subsequent loaders",
			loaders: []configloader.ConfigLoader{
				newMockConfigLoader(MockConfig{
					configv1.ConfigNameJobController: {
						"defaultTTLSecondsAfterFinished": 180,
					},
				}),
				newMockConfigLoader(MockConfig{
					configv1.ConfigNameJobController: {
						"defaultTTLSecondsAfterFinished": 190,
						"defaultPendingTimeoutSeconds":   900,
					},
				}),
				newMockConfigLoader(MockConfig{
					configv1.ConfigNameJobController: {
						"deleteKillingTasksTimeoutSeconds": 30,
					},
				}),
			},
			want: &configv1.JobControllerConfig{
				DefaultTTLSecondsAfterFinished:   190,
				DefaultPendingTimeoutSeconds:     900,
				DeleteKillingTasksTimeoutSeconds: 30,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mgr := configloader.NewConfigManager()
			for _, loader := range tt.loaders {
				mgr.AddConfigLoaders(loader)
			}
			if err := mgr.Start(context.Background()); err != nil {
				t.Fatalf("cannot start ConfigManager: %v", err)
			}

			checkJobControllerConfig(t, mgr, tt.want, tt.wantErr)
		})
	}
}

func TestConfigManager_Dynamic(t *testing.T) {
	type update struct {
		name    string
		config  MockConfig
		want    *configv1.JobControllerConfig
		wantErr bool
	}
	tests := []struct {
		name    string
		initial update
		updates []update
	}{
		{
			name: "dynamic config loader will update",
			initial: update{
				name: "initial value value",
				config: MockConfig{
					configv1.ConfigNameJobController: {
						"defaultTTLSecondsAfterFinished": 180,
					},
				},
				want: &configv1.JobControllerConfig{
					DefaultTTLSecondsAfterFinished: 180,
				},
			},
			updates: []update{
				{
					name: "correct update",
					config: MockConfig{
						configv1.ConfigNameJobController: {
							"defaultTTLSecondsAfterFinished": 181,
						},
					},
					want: &configv1.JobControllerConfig{
						DefaultTTLSecondsAfterFinished: 181,
					},
				},
			},
		},
		{
			name: "invalid initial config but fixed with update",
			initial: update{
				name: "invalid initial config",
				config: MockConfig{
					configv1.ConfigNameJobController: {
						"defaultTTLSecondsAfterFinished": "hello",
					},
				},
				wantErr: true,
			},
			updates: []update{
				{
					name: "return error with another invalid update",
					config: MockConfig{
						configv1.ConfigNameJobController: {
							"defaultTTLSecondsAfterFinished": "hello 2",
						},
					},
					wantErr: true,
				},
				{
					name: "fixed with valid update",
					config: MockConfig{
						configv1.ConfigNameJobController: {
							"defaultTTLSecondsAfterFinished": 180,
						},
					},
					want: &configv1.JobControllerConfig{
						DefaultTTLSecondsAfterFinished: 180,
					},
				},
			},
		},
		{
			name: "valid initial config but invalid after update, should return stale value until fixed",
			initial: update{
				name: "valid initial config",
				config: MockConfig{
					configv1.ConfigNameJobController: {
						"defaultTTLSecondsAfterFinished": 180,
						"defaultPendingTimeoutSeconds":   900,
					},
				},
				want: &configv1.JobControllerConfig{
					DefaultTTLSecondsAfterFinished: 180,
					DefaultPendingTimeoutSeconds:   900,
				},
			},
			updates: []update{
				{
					name: "invalid update",
					config: MockConfig{
						configv1.ConfigNameJobController: {
							"defaultTTLSecondsAfterFinished": "hello",
							"defaultPendingTimeoutSeconds":   900,
						},
					},
					want: &configv1.JobControllerConfig{
						DefaultTTLSecondsAfterFinished: 180,
						DefaultPendingTimeoutSeconds:   900,
					},
				},
				{
					name: "still cannot decode previously value, other valid fields will be stale",
					config: MockConfig{
						configv1.ConfigNameJobController: {
							"defaultTTLSecondsAfterFinished": "hello",
							"defaultPendingTimeoutSeconds":   901,
						},
					},
					want: &configv1.JobControllerConfig{
						DefaultTTLSecondsAfterFinished: 180,
						DefaultPendingTimeoutSeconds:   900,
					},
				},
				{
					name: "finally fixed",
					config: MockConfig{
						configv1.ConfigNameJobController: {
							"defaultTTLSecondsAfterFinished": 181,
							"defaultPendingTimeoutSeconds":   901,
						},
					},
					want: &configv1.JobControllerConfig{
						DefaultTTLSecondsAfterFinished: 181,
						DefaultPendingTimeoutSeconds:   901,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mgr := configloader.NewConfigManager()
			loader := newMockDynamicConfigLoader(tt.initial.config)
			mgr.AddConfigLoaders(loader)
			if err := mgr.Start(context.Background()); err != nil {
				t.Fatalf("cannot start ConfigManager: %v", err)
			}
			checkJobControllerConfig(t, mgr, tt.initial.want, tt.initial.wantErr)

			for i, update := range tt.updates {
				update := update
				t.Run(fmt.Sprintf("step #%02d: %v", i+1, update.name), func(t *testing.T) {
					loader.SetConfig(update.config)
					checkJobControllerConfig(t, mgr, update.want, update.wantErr)
				})
			}
		})
	}
}

func loadJobControllerConfig(mgr *configloader.ConfigManager) (*configv1.JobControllerConfig, error) {
	var config configv1.JobControllerConfig
	if err := mgr.LoadAndUnmarshalConfig(configv1.ConfigNameJobController, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func loadCronControllerConfig(mgr *configloader.ConfigManager) (*configv1.CronControllerConfig, error) {
	var config configv1.CronControllerConfig
	if err := mgr.LoadAndUnmarshalConfig(configv1.ConfigNameCronController, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func checkJobControllerConfig(
	t *testing.T,
	mgr *configloader.ConfigManager,
	want *configv1.JobControllerConfig,
	wantErr bool,
) {
	cfg, err := loadJobControllerConfig(mgr)
	if (err != nil) != wantErr {
		t.Errorf("ControllerConfiguration() want error = %v, got error %v", wantErr, err)
	}
	if err == nil && !cmp.Equal(want, cfg, cmpopts.EquateEmpty()) {
		t.Errorf("ControllerConfiguration() not equal, diff = %v", cmp.Diff(want, cfg))
	}
}
