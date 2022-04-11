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
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/leaderelection"
)

// Controller encapsulates a routine that can be run and shut down.
type Controller interface {
	Runnable

	// Run should block until it terminates; if there is an error, it should return
	// one. If the context is canceled, it should return the context's error in a
	// timely fashion.
	Run(ctx context.Context) error

	// GetHealth returns the health status of the Controller.
	GetHealth() HealthStatus
}

// ControllerManager performs a high-level management of multiple controllers and also
// optionally performs leader election.
type ControllerManager struct {
	*BaseManager
	controllers []Controller
	stores      []Store
	coordinator leaderelection.Coordinator
}

func NewControllerManager(
	ctrlContext controllercontext.Context,
	ctrlCfg configv1alpha1.ControllerManagerConfigSpec,
	defaultLeaseName string,
) (*ControllerManager, error) {
	m := &ControllerManager{
		BaseManager: NewBaseManager(ctrlContext),
	}

	// Enable leader election.
	if cfg := ctrlCfg.LeaderElection; cfg != nil {
		var enabled bool
		if cfg := cfg.Enabled; cfg != nil {
			enabled = *cfg
		}

		if enabled {
			name := cfg.LeaseName
			if name == "" {
				name = defaultLeaseName
			}

			client := ctrlContext.Clientsets().Kubernetes().CoordinationV1()
			coordinator, err := leaderelection.NewCoordinator(client, &leaderelection.Config{
				LeaseName:      name,
				LeaseNamespace: cfg.LeaseNamespace,
				LeaseDuration:  cfg.LeaseDuration.Duration,
				RenewDeadline:  cfg.RenewDeadline.Duration,
				RetryPeriod:    cfg.RetryPeriod.Duration,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "cannot initialize election coordinator")
			}
			m.coordinator = coordinator
		}
	}

	return m, nil
}

// Add a new runnable to the controller manager to be managed. Will not take
// effect once Start is called.
func (m *ControllerManager) Add(runnables ...Runnable) {
	m.BaseManager.Add(runnables...)

	for _, runnable := range runnables {
		if c, ok := runnable.(Controller); ok {
			m.controllers = append(m.controllers, c)
		}
	}
}

// AddStore adds a new Store to be recovered. Will not take effect once Start is
// called.
func (m *ControllerManager) AddStore(stores ...Store) {
	m.stores = append(m.stores, stores...)
}

// Start will start up all controllers and blocks until all of them are started
// and ready. This method will first block until elected, and once elected it
// will block until the controllers are fully started.
//
// If there is an error in starting, this method returns an error. When the
// context is canceled, all controllers should stop their start process as well.
func (m *ControllerManager) Start(ctx context.Context, startTimeout time.Duration) error {
	klog.Infof("controllermanager: starting controller manager")

	// Start the base manager.
	if err := m.BaseManager.Start(ctx); err != nil {
		return err
	}

	// We only let the controller manager pass the readiness probe at this phase
	// once all pre-flight checks are done.
	atomic.StoreUint64(&m.readiness, 1)

	// Start election and block until we are elected.
	if m.coordinator != nil {
		leaderelection.ObserveLeaderElected(m.coordinator.GetLeaseName(), m.coordinator.GetLeaseID(), false)
		klog.Infof("controllermanager: waiting to be elected")
		if err := m.coordinator.Wait(ctx); err != nil {
			return err
		}
		klog.Infof("controllermanager: became elected as leader")
		leaderelection.ObserveLeaderElected(m.coordinator.GetLeaseName(), m.coordinator.GetLeaseID(), true)
	}

	atomic.StoreUint64(&m.starting, 1)

	// Begin recovery and starting all controllers within a timeout.
	if startTimeout > 0 {
		newCtx, cancel := context.WithTimeout(ctx, startTimeout)
		defer cancel()
		ctx = newCtx
	}

	// Recover stores.
	if len(m.stores) > 0 {
		klog.Infof("controllermanager: recovering stores")
		if err := RecoverStores(ctx, m.stores); err != nil {
			return errors.Wrapf(err, "cannot recover stores")
		}
		klog.Infof("controllermanager: recovered all stores")
	}

	startTime := time.Now()
	klog.Infof("controllermanager: starting controllers")
	if err := RunControllers(ctx, m.controllers); err != nil {
		return errors.Wrapf(err, "cannot start controllers")
	}

	atomic.StoreUint64(&m.started, 1)
	klog.Infof("controllermanager: controllers started in %v", time.Since(startTime))

	return nil
}

// GetHealth returns a list of all controllers' health statuses.
func (m *ControllerManager) GetHealth() []HealthStatus {
	healths := make([]HealthStatus, 0, len(m.controllers))

	// If not yet starting (i.e. not elected), do not return false unhealthy state.
	if !m.IsStarting() {
		return healths
	}

	// Get health status of each controller.
	for _, controller := range m.controllers {
		healths = append(healths, controller.GetHealth())
	}
	return healths
}

func (m *ControllerManager) ShutdownAndWait(ctx context.Context) {
	klog.Infof("controllermanager: shutting down")

	// Shut down all runnables.
	m.BaseManager.ShutdownAndWait(ctx)

	// Only give up lease once all controllers have been shut down fully.
	if m.coordinator != nil {
		m.coordinator.GiveUp()
		leaderelection.ObserveLeaderElected(m.coordinator.GetLeaseName(), m.coordinator.GetLeaseID(), false)
	}

	klog.Infof("controllermanager: shut down complete")
}

// RunControllers starts all the given Controllers in the background and blocks
// until all of them are started. If any controller's Run method returns an
// error, this method returns and cancels all other runs. if the context is
// canceled, it will cancel starting up.
func RunControllers(ctx context.Context, controllers []Controller) error {
	grp, _ := errgroup.WithContext(ctx)

	for _, controller := range controllers {
		controller := controller
		grp.Go(func() error {
			return controller.Run(ctx)
		})
	}

	return grp.Wait()
}

// ShutdownRunnables stops all runnables and blocks until they return.
func ShutdownRunnables(ctx context.Context, controllers []Runnable) {
	wg := &sync.WaitGroup{}
	wg.Add(len(controllers))

	for _, controller := range controllers {
		controller := controller
		go func() {
			defer wg.Done()
			controller.Shutdown(ctx)
		}()
	}

	wg.Wait()
}
