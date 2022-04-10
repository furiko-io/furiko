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

package controllermanager

import (
	"context"

	"github.com/pkg/errors"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

type Runnable interface {
	// Shutdown should block until it terminates. This is to provide graceful
	// shutdown.
	Shutdown(ctx context.Context)
}

type HealthStatus struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

// BaseManager is the base manager that manages runnables, readiness and
// liveness status.
type BaseManager struct {
	ctrlContext controllercontext.Context
	readiness   uint64
	starting    uint64
	started     uint64
	runnables   []Runnable
}

func NewBaseManager(ctrlContext controllercontext.Context) *BaseManager {
	return &BaseManager{
		ctrlContext: ctrlContext,
	}
}

// Add a new runnable to the manager to be managed. Will not take
// effect once Start is called.
func (m *BaseManager) Add(runnables ...Runnable) {
	m.runnables = append(m.runnables, runnables...)
}

// Start will perform shared start routines.
func (m *BaseManager) Start(ctx context.Context) error {
	// Start the context.
	if err := m.ctrlContext.Start(ctx); err != nil {
		return errors.Wrapf(err, "cannot start context")
	}

	// Perform initial test to ensure that dynamic configuration is valid and
	// readable. Since the dynamic config manager is able to cache a last known good
	// copy of the config even when the remote config is garbled, we want to ensure
	// that we have at least one good copy of each config in the cache before
	// starting.
	if _, err := m.ctrlContext.Configs().AllConfigs(); err != nil {
		return errors.Wrapf(err, "cannot load dynamic configs")
	}

	return nil
}

// GetReadiness returns an error if the base manager is not yet fully initialized.
func (m *BaseManager) GetReadiness() error {
	if m.readiness == 0 {
		return errors.New("manager not fully initialized")
	}
	return nil
}

// GetHealth returns a list of all health statuses.
// This has to be overridden, and returns an empty slice by default.
func (m *BaseManager) GetHealth() []HealthStatus {
	return []HealthStatus{}
}

func (m *BaseManager) ShutdownAndWait(ctx context.Context) {
	ShutdownRunnables(ctx, m.runnables)
}

func (m *BaseManager) IsStarting() bool {
	return m.starting != 0
}

func (m *BaseManager) IsStarted() bool {
	return m.started != 0
}
