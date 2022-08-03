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

package controllermanager_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/utils/errors"
)

type mockRunnable struct {
	runError        bool
	startupDuration time.Duration
}

func (m mockRunnable) Run(ctx context.Context) error {
	t := time.NewTimer(m.startupDuration)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
	}
	if m.runError {
		return assert.AnError
	}
	return nil
}

type mockController struct {
	name             string
	mock             mockRunnable
	shutdownDuration time.Duration
	shutdownDone     uint64
}

func newMockController(name string, mock mockRunnable) *mockController {
	return &mockController{
		name: name,
		mock: mock,
	}
}

var _ controllermanager.Controller = (*mockController)(nil)

func (c *mockController) Run(ctx context.Context) error {
	return c.mock.Run(ctx)
}

func (c *mockController) GetHealth() controllermanager.HealthStatus {
	return controllermanager.HealthStatus{
		Name:    c.name,
		Healthy: true,
	}
}

func (c *mockController) Shutdown(ctx context.Context) {
	t := time.NewTimer(c.shutdownDuration)
	select {
	case <-ctx.Done():
		return
	case <-t.C:
	}
	atomic.StoreUint64(&c.shutdownDone, 1)
}

type mockStore struct {
	name string
	mock mockRunnable
}

var _ controllermanager.Store = (*mockStore)(nil)

func newMockStore(name string, mock mockRunnable) *mockStore {
	return &mockStore{
		name: name,
		mock: mock,
	}
}

func (m *mockStore) Name() string {
	return m.name
}

func (m *mockStore) Recover(ctx context.Context) error {
	return m.mock.Run(ctx)
}

func TestControllerManager_Start(t *testing.T) {
	tests := []struct {
		name             string
		ctrlCfg          configv1alpha1.ControllerManagerConfigSpec
		defaultLeaseName string
		controllers      []controllermanager.Controller
		stores           []controllermanager.Store
		wantInitErr      bool
		timeout          time.Duration
		cancelAfter      time.Duration
		wantStartErr     assert.ErrorAssertionFunc
	}{
		{
			name:         "no controllers",
			wantStartErr: assert.NoError,
		},
		{
			name: "start single controller",
			controllers: []controllermanager.Controller{
				newMockController("mock", mockRunnable{}),
			},
			wantStartErr: assert.NoError,
		},
		{
			name: "start single controller with error",
			controllers: []controllermanager.Controller{
				newMockController("mock", mockRunnable{runError: true}),
			},
			wantStartErr: errors.AssertIs(assert.AnError),
		},
		{
			name: "start single store",
			stores: []controllermanager.Store{
				newMockStore("mock", mockRunnable{}),
			},
			wantStartErr: assert.NoError,
		},
		{
			name: "start single store with error",
			stores: []controllermanager.Store{
				newMockStore("mock", mockRunnable{runError: true}),
			},
			wantStartErr: errors.AssertIs(assert.AnError),
		},
		{
			name: "start controller timeout",
			controllers: []controllermanager.Controller{
				newMockController("mock1", mockRunnable{startupDuration: time.Millisecond * 500}),
				newMockController("mock2", mockRunnable{startupDuration: time.Millisecond * 20}),
			},
			timeout:      time.Millisecond * 50,
			wantStartErr: errors.AssertIs(context.DeadlineExceeded),
		},
		{
			name: "start store timeout",
			controllers: []controllermanager.Controller{
				newMockController("mock1", mockRunnable{}),
				newMockController("mock2", mockRunnable{}),
			},
			stores: []controllermanager.Store{
				newMockStore("mock", mockRunnable{startupDuration: time.Millisecond * 500}),
			},
			timeout:      time.Millisecond * 50,
			wantStartErr: errors.AssertIs(context.DeadlineExceeded),
		},
		{
			name: "start multiple controllers and stores with varying duration",
			controllers: []controllermanager.Controller{
				newMockController("mock1", mockRunnable{startupDuration: time.Millisecond * 50}),
				newMockController("mock2", mockRunnable{startupDuration: time.Millisecond * 20}),
			},
			stores: []controllermanager.Store{
				newMockStore("mock1", mockRunnable{startupDuration: time.Millisecond * 30}),
				newMockStore("mock2", mockRunnable{startupDuration: time.Millisecond * 60}),
			},
			timeout:      time.Second,
			wantStartErr: assert.NoError,
		},
		{
			name: "cancel context while starting",
			controllers: []controllermanager.Controller{
				newMockController("mock1", mockRunnable{startupDuration: time.Millisecond * 50}),
				newMockController("mock2", mockRunnable{startupDuration: time.Millisecond * 20}),
			},
			stores: []controllermanager.Store{
				newMockStore("mock1", mockRunnable{startupDuration: time.Millisecond * 30}),
				newMockStore("mock2", mockRunnable{startupDuration: time.Millisecond * 60}),
			},
			cancelAfter:  time.Millisecond * 40,
			timeout:      time.Second,
			wantStartErr: errors.AssertIs(context.DeadlineExceeded),
		},
		{
			name: "start with leader election",
			ctrlCfg: configv1alpha1.ControllerManagerConfigSpec{
				LeaderElection: &configv1alpha1.LeaderElectionSpec{
					Enabled: pointer.Bool(true),
				},
			},
			defaultLeaseName: "execution-controller",
			controllers: []controllermanager.Controller{
				newMockController("mock", mockRunnable{}),
			},
			wantStartErr: assert.NoError,
		},
		{
			name: "need default lease name",
			ctrlCfg: configv1alpha1.ControllerManagerConfigSpec{
				LeaderElection: &configv1alpha1.LeaderElectionSpec{
					Enabled: pointer.Bool(true),
				},
			},
			controllers: []controllermanager.Controller{
				newMockController("mock", mockRunnable{}),
			},
			wantInitErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mock.NewContext()
			mgr, err := controllermanager.NewControllerManager(c, tt.ctrlCfg, tt.defaultLeaseName)
			if (err != nil) != tt.wantInitErr {
				t.Errorf("NewControllerManager(c, %v, %v) got err %v, wantInitErr = %v",
					tt.ctrlCfg, tt.defaultLeaseName, err, tt.wantInitErr)
			}
			if err != nil {
				return
			}
			for _, controller := range tt.controllers {
				mgr.Add(controller)
			}
			mgr.AddStore(tt.stores...)
			ctx := context.Background()
			if tt.cancelAfter > 0 {
				newCtx, cancel := context.WithTimeout(context.Background(), tt.cancelAfter)
				defer cancel()
				ctx = newCtx
			}
			tt.wantStartErr(t, mgr.Start(ctx, tt.timeout))
		})
	}
}

func TestControllerManager_Shutdown(t *testing.T) {
	tests := []struct {
		name          string
		controllers   []*mockController
		cancelAfter   time.Duration
		wantInterrupt bool
	}{
		{
			name: "no controllers",
		},
		{
			name: "shutdown without delay",
			controllers: []*mockController{
				newMockController("mock", mockRunnable{}),
			},
		},
		{
			name: "shutdown interrupted",
			controllers: []*mockController{
				newMockController("mock1", mockRunnable{}),
				{
					name:             "mock2",
					shutdownDuration: time.Millisecond * 500,
				},
			},
			cancelAfter:   time.Millisecond * 50,
			wantInterrupt: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mock.NewContext()
			mgr, err := controllermanager.NewControllerManager(c, configv1alpha1.ControllerManagerConfigSpec{}, "")
			assert.NoError(t, err)
			for _, controller := range tt.controllers {
				mgr.Add(controller)
			}
			err = mgr.Start(context.Background(), 0)
			assert.NoError(t, err)

			ctx := context.Background()
			if tt.cancelAfter > 0 {
				newCtx, cancel := context.WithTimeout(context.Background(), tt.cancelAfter)
				defer cancel()
				ctx = newCtx
			}
			mgr.ShutdownAndWait(ctx)
			allDone := true
			hasInterrupt := false
			for _, controller := range tt.controllers {
				if controller.shutdownDone == 0 {
					hasInterrupt = true
					allDone = false
				}
			}
			if tt.wantInterrupt {
				assert.True(t, hasInterrupt)
			} else {
				assert.True(t, allDone)
			}
		})
	}
}

func TestControllerManager_GetHealth(t *testing.T) {
	c := mock.NewContext()
	mgr, err := controllermanager.NewControllerManager(c, configv1alpha1.ControllerManagerConfigSpec{}, "")
	assert.NoError(t, err)

	// Add some controllers
	mgr.Add(newMockController("mock1", mockRunnable{}))
	mgr.Add(newMockController("mock2", mockRunnable{}))

	// No health status before starting.
	assert.Len(t, mgr.GetHealth(), 0)

	// Populate health status after starting.
	err = mgr.Start(context.Background(), 0)
	assert.NoError(t, err)
	assert.Len(t, mgr.GetHealth(), 2)
}
