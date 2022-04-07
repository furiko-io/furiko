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
	"k8s.io/utils/pointer"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	"github.com/furiko-io/furiko/pkg/runtime/configloader"
)

const (
	ConfigName = "mock-config"
)

type Config struct {
	// Fields from JobControllerConfig.
	DefaultTTLSecondsAfterFinished   int64 `json:"defaultTTLSecondsAfterFinished,omitempty"`
	DefaultPendingTimeoutSeconds     int64 `json:"defaultPendingTimeoutSeconds,omitempty"`
	DeleteKillingTasksTimeoutSeconds int64 `json:"deleteKillingTasksTimeoutSeconds,omitempty"`

	// Fields for type-specific tests.
	Bool    bool  `json:"bool,omitempty"`
	BoolPtr *bool `json:"boolPtr,omitempty"`
	Int     int   `json:"int,omitempty"`
	IntPtr  *int  `json:"intPtr,omitempty"`
}

type MockConfig map[configv1.ConfigName]configloader.Config

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

func (m *mockErrorLoader) Load(_ configv1.ConfigName) (configloader.Config, error) {
	return nil, errors.New("load config error")
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

func (m *mockConfigLoader) Load(configName configv1.ConfigName) (configloader.Config, error) {
	return m.values[configName], nil
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
		loaders []configloader.Loader
		want    *Config
		wantErr bool
	}{
		{
			name:    "no loaders",
			loaders: nil,
			want:    &Config{},
		},
		{
			name: "loader error",
			loaders: []configloader.Loader{
				newMockErrorLoader(),
			},
			wantErr: true,
		},
		{
			name: "single loader",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"defaultTTLSecondsAfterFinished": 180,
						"defaultPendingTimeoutSeconds":   900,
					},
				}),
			},
			want: &Config{
				DefaultTTLSecondsAfterFinished: 180,
				DefaultPendingTimeoutSeconds:   900,
			},
		},
		{
			name: "override values and add new fields in subsequent loaders",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"defaultTTLSecondsAfterFinished": 180,
					},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"defaultTTLSecondsAfterFinished": 190,
						"defaultPendingTimeoutSeconds":   900,
					},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"deleteKillingTasksTimeoutSeconds": 30,
					},
				}),
			},
			want: &Config{
				DefaultTTLSecondsAfterFinished:   190,
				DefaultPendingTimeoutSeconds:     900,
				DeleteKillingTasksTimeoutSeconds: 30,
			},
		},
		{
			name: "nil pointers",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"intPtr":  nil,
						"boolPtr": nil,
					},
				}),
			},
			want: &Config{},
		},
		{
			name: "override false bool pointer with true",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"boolPtr": false,
					},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"boolPtr": true,
					},
				}),
			},
			want: &Config{
				BoolPtr: pointer.Bool(true),
			},
		},
		{
			name: "override true bool pointer with false",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"boolPtr": true,
					},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"boolPtr": false,
					},
				}),
			},
			want: &Config{
				BoolPtr: pointer.Bool(false),
			},
		},
		{
			name: "do not override true bool pointer with nil",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"boolPtr": true,
					},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {},
				}),
			},
			want: &Config{
				BoolPtr: pointer.Bool(true),
			},
		},
		{
			name: "do not override false bool pointer with nil",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"boolPtr": false,
					},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {},
				}),
			},
			want: &Config{
				BoolPtr: pointer.Bool(false),
			},
		},
		{
			name: "override nil bool pointer with true",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"boolPtr": true,
					},
				}),
			},
			want: &Config{
				BoolPtr: pointer.Bool(true),
			},
		},
		{
			name: "override nil bool pointer with false",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"boolPtr": false,
					},
				}),
			},
			want: &Config{
				BoolPtr: pointer.Bool(false),
			},
		},
		{
			name: "do not override int pointers with nil",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"intPtr": 100,
					},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {},
				}),
			},
			want: &Config{
				IntPtr: pointer.Int(100),
			},
		},
		{
			name: "override int pointers",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"intPtr": 100,
					},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"intPtr": 101,
					},
				}),
			},
			want: &Config{
				IntPtr: pointer.Int(101),
			},
		},
		{
			name: "override int pointers with 0",
			loaders: []configloader.Loader{
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"intPtr": 100,
					},
				}),
				newMockConfigLoader(MockConfig{
					ConfigName: {
						"intPtr": 0,
					},
				}),
			},
			want: &Config{
				IntPtr: pointer.Int(0),
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

			checkConfig(t, mgr, tt.want, tt.wantErr)
		})
	}
}

func TestConfigManager_Dynamic(t *testing.T) {
	type update struct {
		name    string
		config  MockConfig
		want    *Config
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
					ConfigName: {
						"defaultTTLSecondsAfterFinished": 180,
					},
				},
				want: &Config{
					DefaultTTLSecondsAfterFinished: 180,
				},
			},
			updates: []update{
				{
					name: "correct update",
					config: MockConfig{
						ConfigName: {
							"defaultTTLSecondsAfterFinished": 181,
						},
					},
					want: &Config{
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
					ConfigName: {
						"defaultTTLSecondsAfterFinished": "hello",
					},
				},
				wantErr: true,
			},
			updates: []update{
				{
					name: "return error with another invalid update",
					config: MockConfig{
						ConfigName: {
							"defaultTTLSecondsAfterFinished": "hello 2",
						},
					},
					wantErr: true,
				},
				{
					name: "fixed with valid update",
					config: MockConfig{
						ConfigName: {
							"defaultTTLSecondsAfterFinished": 180,
						},
					},
					want: &Config{
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
					ConfigName: {
						"defaultTTLSecondsAfterFinished": 180,
						"defaultPendingTimeoutSeconds":   900,
					},
				},
				want: &Config{
					DefaultTTLSecondsAfterFinished: 180,
					DefaultPendingTimeoutSeconds:   900,
				},
			},
			updates: []update{
				{
					name: "invalid update",
					config: MockConfig{
						ConfigName: {
							"defaultTTLSecondsAfterFinished": "hello",
							"defaultPendingTimeoutSeconds":   900,
						},
					},
					want: &Config{
						DefaultTTLSecondsAfterFinished: 180,
						DefaultPendingTimeoutSeconds:   900,
					},
				},
				{
					name: "still cannot decode previously value, other valid fields will be stale",
					config: MockConfig{
						ConfigName: {
							"defaultTTLSecondsAfterFinished": "hello",
							"defaultPendingTimeoutSeconds":   901,
						},
					},
					want: &Config{
						DefaultTTLSecondsAfterFinished: 180,
						DefaultPendingTimeoutSeconds:   900,
					},
				},
				{
					name: "finally fixed",
					config: MockConfig{
						ConfigName: {
							"defaultTTLSecondsAfterFinished": 181,
							"defaultPendingTimeoutSeconds":   901,
						},
					},
					want: &Config{
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
			checkConfig(t, mgr, tt.initial.want, tt.initial.wantErr)

			for i, update := range tt.updates {
				update := update
				t.Run(fmt.Sprintf("step #%02d: %v", i+1, update.name), func(t *testing.T) {
					loader.SetConfig(update.config)
					checkConfig(t, mgr, update.want, update.wantErr)
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

func checkConfig(t *testing.T, mgr *configloader.ConfigManager, want *Config, wantErr bool) {
	cfg := &Config{}
	err := mgr.LoadAndUnmarshalConfig(ConfigName, cfg)
	if (err != nil) != wantErr {
		t.Errorf("LoadAndUnmarshalConfig() want error = %v, got error %v", wantErr, err)
	}
	if err == nil && !cmp.Equal(want, cfg, cmpopts.EquateEmpty()) {
		t.Errorf("LoadAndUnmarshalConfig() not equal, diff = %v", cmp.Diff(want, cfg))
	}
}
