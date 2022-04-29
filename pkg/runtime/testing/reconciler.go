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

package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	testinginterface "github.com/mitchellh/go-testing-interface"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

const (
	defaultReconcilerTestTimeout = time.Second
)

// ControllerContext is implemented by controller contexts used in reconciler tests.
type ControllerContext interface {
	controllercontext.Context
	GetHasSynced() []cache.InformerSynced
}

// ReconcilerTest encapsulates multiple test cases, providing a standard
// framework for testing standard Reconcilers.
type ReconcilerTest struct {
	// Returns a new controller context.
	ContextFunc func(c controllercontext.Context, recorder record.EventRecorder) ControllerContext

	// Returns the reconciler to test and informer synced methods to wait for.
	ReconcilerFunc func(c ControllerContext) reconciler.Reconciler

	// Sets the current time.
	Now time.Time

	// Specifies the timeout for each test case. Defaults to 1 second.
	Timeout time.Duration

	// Stores to create.
	Stores []mock.StoreFactory
}

// ReconcilerTestCase encapsulates a single Reconciler test case, including
// fixtures, mocks, target to be synced and expected actions.
type ReconcilerTestCase struct {
	// Name of the test case.
	Name string

	// Target to be synced. It will be created before syncing.
	Target runtime.Object

	// Generator function that returns the Target. This generator will be called
	// lazily, evaluated only after all initialization is done. For example, if the
	// target depends on the clock to have already been mocked, use TargetGenerator
	// instead of Target.
	TargetGenerator func() runtime.Object

	// SyncTarget explicitly specifies the namespace and name to sync.
	// If left empty, will derive from Target.
	SyncTarget *SyncTarget

	// Other fixtures to be created prior to the sync.
	Fixtures []runtime.Object

	// Optionally specifies the current time to mock to override from the parent
	// ReconcilerTest.
	Now time.Time

	// Optionally specifies configs to set.
	Configs controllercontext.ConfigsMap

	// Optional list of Reactors, which intercepts clientset actions.
	Reactors CombinedReactors

	// Defines additional assertion functions.
	Assert func(t assert.TestingT, tt ReconcilerTestCase, ctrlContext ControllerContext)

	// List of actions that we expect to see.
	WantActions CombinedActions

	// List of events that we expect to see.
	WantEvents []Event

	// Whether an error is expected, and if so, specifies a function to check if the
	// error is equal.
	WantError assert.ErrorAssertionFunc
}

type CombinedReactors struct {
	// List of reactors for the kubernetes clientset.
	Kubernetes []*ktesting.SimpleReactor

	// List of reactors for the furiko clientset.
	Furiko []*ktesting.SimpleReactor
}

type CombinedActions struct {
	// List of actions that are expected on the kubernetes clientset.
	Kubernetes ActionTest

	// List of actions that are expected on the furiko clientset.
	Furiko ActionTest
}

type SyncTarget struct {
	Namespace string
	Name      string
}

// Run executes all test cases.
func (r *ReconcilerTest) Run(t *testing.T, cases []ReconcilerTestCase) {
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			r.RunTestCase(t, tt)
		})
	}
}

// RunTestCase executes a single ReconcilerTestCase.
func (r *ReconcilerTest) RunTestCase(t testinginterface.T, tt ReconcilerTestCase) {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultReconcilerTestTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Initialize context.
	c := mock.NewContext()
	ctrlContext, recorder := r.setupContext(c, tt)

	// Initialize reconciler.
	recon := r.ReconcilerFunc(ctrlContext)

	// Start the context.
	err := c.Start(ctx)
	assert.NoError(t, err)

	// Initialize fixtures.
	target, err := r.initClientset(ctx, c.MockClientsets(), tt, ctrlContext.GetHasSynced())
	if err != nil {
		t.Fatalf("cannot initialize fixtures: %v", err)
	}

	// Trigger sync.
	if err := r.triggerReconcile(ctx, t, target, tt, recon); err != nil {
		t.Fatalf("cannot trigger reconcile: %v", err)
	}

	// Compare actions.
	r.assertResult(t, tt, c.MockClientsets(), ctrlContext, recorder)
}

func (r *ReconcilerTest) setupContext(
	c *mock.Context,
	tt ReconcilerTestCase,
) (ControllerContext, *FakeRecorder) {
	// Set up configs.
	c.MockConfigs().SetConfigs(tt.Configs)

	// Set up stores.
	c.MockStores().RegisterFromFactoriesOrDie(r.Stores...)

	// Set up clock.
	now := r.Now
	if !tt.Now.IsZero() {
		now = tt.Now
	}
	fakeClock := clock.NewFakeClock(now)
	ktime.Clock = fakeClock

	// Set up fake recorder.
	recorder := NewFakeRecorder()

	// Initialize context.
	return r.ContextFunc(c, recorder), recorder
}

// initClientset performs initialization of all fixtures and reactors with the given clientsets.
func (r *ReconcilerTest) initClientset(
	ctx context.Context,
	client *mock.Clientsets,
	tt ReconcilerTestCase,
	hasSynced []cache.InformerSynced,
) (runtime.Object, error) {
	// Set up all fixtures.
	if err := InitFixtures(ctx, client, tt.Fixtures); err != nil {
		return nil, err
	}

	// Set up target fixture.
	target := tt.Target
	if tt.TargetGenerator != nil {
		target = tt.TargetGenerator()
	}
	if target != nil {
		if err := InitFixture(ctx, client, target); err != nil {
			return nil, errors.Wrapf(err, "cannot initialize target fixture")
		}
	}

	// Set up reactors.
	for _, reactor := range tt.Reactors.Kubernetes {
		client.KubernetesMock().PrependReactor(reactor.Verb, reactor.Resource, reactor.React)
	}
	for _, reactor := range tt.Reactors.Furiko {
		client.FurikoMock().PrependReactor(reactor.Verb, reactor.Resource, reactor.React)
	}

	// Wait for cache sync
	// NOTE(irvinlim): Add a short delay otherwise cache may not sync consistently
	time.Sleep(time.Millisecond * 50)
	if !cache.WaitForCacheSync(ctx.Done(), hasSynced...) {
		return nil, errors.New("caches not synced")
	}

	// Clear all actions prior to reconcile.
	client.ClearActions()

	return target, nil
}

func (r *ReconcilerTest) triggerReconcile(
	ctx context.Context,
	t testinginterface.T,
	target interface{},
	tt ReconcilerTestCase,
	recon reconciler.Reconciler,
) error {
	// Get sync target.
	syncTarget := tt.SyncTarget
	if syncTarget == nil {
		key, err := cache.MetaNamespaceKeyFunc(target)
		if err != nil {
			return errors.Wrapf(err, "cannot create namespaced key for target %v", target)
		}
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return errors.Wrapf(err, "cannot split namespace and name from key: %v", key)
		}
		syncTarget = &SyncTarget{
			Namespace: namespace,
			Name:      name,
		}
	}

	// Trigger the reconciler.
	wantError := tt.WantError
	if wantError == nil {
		wantError = assert.NoError
	}
	wantError(t, recon.SyncOne(ctx, syncTarget.Namespace, syncTarget.Name, 0),
		fmt.Sprintf("Error in test %v", tt.Name))

	return nil
}

func (r *ReconcilerTest) assertResult(
	t testinginterface.T,
	tt ReconcilerTestCase,
	client *mock.Clientsets,
	ctrlContext ControllerContext,
	recorder *FakeRecorder,
) {
	// Compare actions.
	CompareActions(t, tt.WantActions.Kubernetes, client.KubernetesMock().Actions())
	CompareActions(t, tt.WantActions.Furiko, client.FurikoMock().Actions())

	// Compare recorded events.
	r.compareEvents(t, recorder.GetEvents(), tt.WantEvents)

	// Run custom assertions.
	if tt.Assert != nil {
		tt.Assert(t, tt, ctrlContext)
	}
}

func (r *ReconcilerTest) compareEvents(t testinginterface.T, events, wantEvents []Event) {
	var idx int
	for _, event := range events {
		if idx >= len(wantEvents) {
			t.Errorf("saw extra event: %v", event)
			continue
		}
		assert.Equal(t, wantEvents[idx], event)
		idx++
	}
	for i := idx; i < len(wantEvents); i++ {
		t.Errorf("did not see event: %v", wantEvents[i])
	}
}
