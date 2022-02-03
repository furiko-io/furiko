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

package leaderelection

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

type Coordinator interface {
	IsLeader() bool
	GetLeader() string
	Wait(ctx context.Context) error
	GiveUp()
	GetLeaseName() string
	GetLeaseID() string
}

// NewCoordinator creates a new Coordinator with a randomly generated identity.
func NewCoordinator(client coordinationv1.LeasesGetter, config *Config) (Coordinator, error) {
	id, err := GenerateLeaseIdentity()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot generate lease identity")
	}
	return NewCoordinatorWithIdentity(client, id, config)
}

// NewCoordinatorWithIdentity creates a new Coordinator with the given identity.
func NewCoordinatorWithIdentity(client coordinationv1.LeasesGetter, id string, config *Config) (Coordinator, error) {
	return newElectionCoordinator(client, id, config)
}

// electionCoordinator coordinates a leader election.
type electionCoordinator struct {
	name            string
	id              string
	elector         *leaderelection.LeaderElector
	waitToBeLeader  chan struct{}
	electionLoopCtx context.Context
	cancelElection  context.CancelFunc
	electionLoopWg  *sync.WaitGroup
}

func newElectionCoordinator(
	client coordinationv1.LeasesGetter, id string, config *Config,
) (*electionCoordinator, error) {
	config = config.PrepareValues()

	if config.LeaseName == "" {
		return nil, errors.New("lease name cannot be empty")
	}

	c := &electionCoordinator{
		waitToBeLeader: make(chan struct{}),
		name:           config.LeaseName,
		id:             id,

		// Election loop control structures.
		// NOTE(irvinlim): If this context is canceled, it means that we
		// want to terminate the election loop.
		electionLoopCtx: context.Background(),
		electionLoopWg:  &sync.WaitGroup{},
	}

	rl := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      config.LeaseName,
			Namespace: config.LeaseNamespace,
		},
		Client: client,
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	lc := leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: config.LeaseDuration,
		RenewDeadline: config.RenewDeadline,
		RetryPeriod:   config.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: c.OnStartedLeading,
			OnStoppedLeading: c.OnStoppedLeading,
		},
		Name: rl.Identity(),

		// We pass our own context to the Run function, so we can cancel it programmatically.
		ReleaseOnCancel: true,
	}

	elector, err := leaderelection.NewLeaderElector(lc)
	if err != nil {
		return nil, err
	}

	c.elector = elector
	return c, nil
}

// IsLeader returns true if the current instance is currently elected as leader.
func (c *electionCoordinator) IsLeader() bool {
	return c.elector.IsLeader()
}

// GetLeader returns the ID of the current leading instance.
func (c *electionCoordinator) GetLeader() string {
	return c.elector.GetLeader()
}

// GetLeaseName returns the lease name that we are using.
func (c *electionCoordinator) GetLeaseName() string {
	return c.name
}

// GetLeaseID returns the lease ID that we are using.
func (c *electionCoordinator) GetLeaseID() string {
	return c.id
}

// Wait starts the election and blocks until elected as leader.
func (c *electionCoordinator) Wait(ctx context.Context) error {
	// Create subcontext.
	ctx, cancel := context.WithCancel(ctx)
	c.electionLoopCtx = ctx
	c.cancelElection = cancel

	klog.InfoS("leaderelection: starting election", "lease", c.name, "lease_id", c.id)

	// Run election loop in goroutine.
	c.electionLoopWg.Add(1)
	go func() {
		defer c.electionLoopWg.Done()

		// Blocks until context is canceled or lease is lost.
		c.elector.Run(ctx)
		klog.V(4).InfoS("leaderelection: election loop done", "lease", c.name, "lease_id", c.id)
	}()

	klog.InfoS("leaderelection: waiting to be leader", "lease", c.name, "lease_id", c.id)

	// Block until elected.
	select {
	case <-c.waitToBeLeader:
		klog.InfoS("leaderelection: got elected as leader", "lease", c.name, "lease_id", c.id)
		return nil
	case <-ctx.Done():
		klog.ErrorS(ctx.Err(), "leaderelection: cancelling participation in election",
			"lease", c.name, "lease_id", c.id)

		// Wait for election loop to be canceled before returning
		c.electionLoopWg.Wait()
		return ctx.Err()
	}
}

// GiveUp will release the lease.
func (c *electionCoordinator) GiveUp() {
	klog.InfoS("leaderelection: giving up lease", "lease", c.name, "lease_id", c.id)

	// We cancel the context that was passed to the election loop, used in
	// conjunction with ReleaseWithCancel.
	if c.cancelElection != nil {
		klog.V(4).Info("leaderelection: cancelling election loop")
		c.cancelElection()
	}

	// Wait for election loop to be canceled. Otherwise, we might not necessarily
	// give up the lease before we return, which may result in having to wait the
	// full lease duration upon starting up.
	c.electionLoopWg.Wait()

	klog.InfoS("leaderelection: gave up lease", "lease", c.name, "lease_id", c.id)
}

// OnStartedLeading coordinates the elected status.
func (c *electionCoordinator) OnStartedLeading(_ context.Context) {
	klog.InfoS("leaderelection: started leading", "lease", c.name, "lease_id", c.id)
	close(c.waitToBeLeader)
}

// OnStoppedLeading is implemented as simply to exit with fatal message.
func (c *electionCoordinator) OnStoppedLeading() {
	// We were leading, but we intentionally gave up the lease.
	select {
	case <-c.electionLoopCtx.Done():
		klog.InfoS("leaderelection: stopped leading", "lease", c.name, "lease_id", c.id)
		return
	default:
	}

	// We were leading but did not give up the lease, and the lease is now gone.
	// Exit the application immediately using a fatal log message, in order to
	// prevent concurrent instances.
	select {
	case <-c.waitToBeLeader:
		klog.InfoS("leaderelection: stopped leading", "lease", c.name, "lease_id", c.id)
		klog.Fatalf("leaderelection: lost the lease, exiting now...")
	default:
	}
}
