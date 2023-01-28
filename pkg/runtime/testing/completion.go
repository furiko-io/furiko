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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclock "k8s.io/utils/clock/testing"

	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

// RunCompleterTests executes all CompleterTest cases.
func RunCompleterTests(t *testing.T, cases []CompleterTest) {
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

// CompleterTest is a single Completer test case to be run.
type CompleterTest struct {
	// Name of the test case.
	Name string

	// Completer under test.
	Completer completion.Completer

	// Fixtures to be created prior to calling the command.
	Fixtures []runtime.Object

	// Sets the current time.
	Now time.Time

	// Namespace to pass to the Completer.
	Namespace string

	// Expected completions, order is important.
	Want []string

	// Whether an error is expected, and if so, specifies a function to check if the
	// error is equal.
	WantError assert.ErrorAssertionFunc
}

// Run the test case.
func (c *CompleterTest) Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Override the shared context.
	ctrlContext := mock.NewContext()
	common.SetCtrlContext(ctrlContext)
	client := ctrlContext.MockClientsets()
	assert.NoError(t, InitFixtures(ctx, client, c.Fixtures))
	client.ClearActions()

	// Set up clock.
	now := time.Now()
	if !c.Now.IsZero() {
		now = c.Now
	}
	ktime.Clock = fakeclock.NewFakeClock(now)

	// Execute the Completer.
	got, err := c.Completer.Complete(ctx, ctrlContext, c.Namespace)
	if testutils.WantError(t, c.WantError, err) {
		return
	}

	// Ensure that output matches.
	if !cmp.Equal(c.Want, got) {
		t.Errorf("Complete() not equal:\ndiff = %v", cmp.Diff(c.Want, got))
	}
}
