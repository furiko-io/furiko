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
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/furiko-io/furiko/pkg/cli/cmd"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
)

// RunCommandTests executes all CommandTest cases.
func RunCommandTests(t *testing.T, cases []CommandTest) {
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

// CommandTest encapsulates a single CLI command test case to be run.
type CommandTest struct {
	// Name of the test case.
	Name string

	// Arguments to be passed to the command.
	Args []string

	// Fixtures to be created prior to calling the command.
	Fixtures []runtime.Object

	// Output rules for standard output.
	Stdout Output

	// Output rules for standard error.
	Stderr Output

	// Whether an error is expected, and if so, specifies a function to check if the
	// error is equal.
	WantError assert.ErrorAssertionFunc
}

type Output struct {
	// If specified, expects the output to match exactly.
	Exact string

	// If specified, expects the output to contain the given string.
	Contains string

	// If specified, expects the output to contain all the given strings.
	ContainsAll []string
}

func (c *CommandTest) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Override the shared context.
	ctrlContext := mock.NewContext()
	cmd.SetCtrlContext(ctrlContext)
	assert.NoError(t, InitFixtures(ctx, ctrlContext.Clientsets(), c.Fixtures))

	// Prepare root command.
	streams, _, stdout, stderr := genericclioptions.NewTestIOStreams()
	command := cmd.NewRootCommand(streams)

	// Set args and execute.
	command.SetArgs(c.Args)
	err := command.Execute()

	// Check for error.
	wantError := c.WantError
	if wantError == nil {
		wantError = assert.NoError
	}
	wantError(t, err, fmt.Sprintf("Run error with args: %v", c.Args))
	if err != nil {
		return
	}

	// Ensure that output matches.
	c.checkOutput(t, "stdout", stdout.String(), c.Stdout)
	c.checkOutput(t, "stderr", stderr.String(), c.Stderr)
}

func (c *CommandTest) checkOutput(t *testing.T, name, s string, output Output) {
	if output.Exact != "" && s != output.Exact {
		t.Errorf("Output in %v not equal\ndiff = %v", name, cmp.Diff(output.Exact, s))
	}

	if output.Contains != "" && !strings.Contains(s, output.Contains) {
		t.Errorf(`Output in %v did not contain expected string "%v", got: %v`, name, output.Contains, s)
	}

	if len(output.ContainsAll) > 0 {
		for _, contains := range output.ContainsAll {
			if !strings.Contains(s, contains) {
				t.Errorf(`Output in %v did not contain expected string "%v", got: %v`, name, contains, s)
			}
		}
	}
}
