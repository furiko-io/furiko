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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/furiko-io/furiko/pkg/cli/cmd"
	"github.com/furiko-io/furiko/pkg/cli/console"
	"github.com/furiko-io/furiko/pkg/cli/streams"
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

	// Input rules for standard input.
	Stdin Input

	// Output rules for standard output.
	Stdout Output

	// Output rules for standard error.
	Stderr Output

	// Whether an error is expected, and if so, specifies a function to check if the
	// error is equal.
	WantError assert.ErrorAssertionFunc

	// List of actions that we expect to see.
	WantActions CombinedActions
}

type Input struct {
	// If specified, the procedure will be called with a pseudo-TTY. The console
	// argument can be used to expect and send values to the TTY.
	Procedure func(console *console.Console)
}

type Output struct {
	// If specified, expects the output to match exactly.
	Exact string

	// If specified, expects the output to contain the given string.
	Contains string

	// If specified, expects the output to contain all the given strings.
	ContainsAll []string

	// If specified, expects the output to match the specified regexp.
	Matches *regexp.Regexp
}

func (c *CommandTest) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Override the shared context.
	ctrlContext := mock.NewContext()
	cmd.SetCtrlContext(ctrlContext)
	client := ctrlContext.MockClientsets()
	assert.NoError(t, InitFixtures(ctx, client, c.Fixtures))
	client.ClearActions()

	// Run command with I/O.
	iostreams, _, stdout, stderr := genericclioptions.NewTestIOStreams()
	if c.runCommand(t, iostreams) {
		return
	}

	// Ensure that output matches.
	c.checkOutput(t, "stdout", stdout.String(), c.Stdout)
	c.checkOutput(t, "stderr", stderr.String(), c.Stderr)

	// Compare actions.
	CompareActions(t, c.WantActions.Kubernetes, client.KubernetesMock().Actions())
	CompareActions(t, c.WantActions.Furiko, client.FurikoMock().Actions())
}

// runCommand will execute the command, setting up all I/O streams and blocking
// until the streams are done.
//
// Reference:
// https://github.com/AlecAivazis/survey/blob/93657ef69381dd1ffc7a4a9cfe5a2aefff4ca4ad/survey_posix_test.go#L15
func (c *CommandTest) runCommand(t *testing.T, iostreams genericclioptions.IOStreams) bool {
	console, err := console.NewConsole(iostreams.Out)
	if err != nil {
		t.Fatalf("failed to create console: %v", err)
	}
	defer console.Close()

	// Prepare root command.
	command := cmd.NewRootCommand(streams.NewTTYStreams(console.Tty()))
	command.SetArgs(c.Args)

	// Pass input to the pty.
	done := console.Run(c.Stdin.Procedure)

	// Execute command.
	if WantError(t, c.WantError, command.Execute(), fmt.Sprintf("Run error with args: %v", c.Args)) {
		return true
	}

	// TODO(irvinlim): We need a sleep here otherwise tests will be flaky
	time.Sleep(time.Millisecond * 5)

	// Wait for tty to be closed.
	if err := console.Tty().Close(); err != nil {
		t.Errorf("error closing Tty: %v", err)
	}
	<-done

	return false
}

func (c *CommandTest) checkOutput(t *testing.T, name, s string, output Output) {
	// Replace all CRLF from PTY back to normal LF.
	s = strings.ReplaceAll(s, "\r\n", "\n")

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

	if output.Matches != nil {
		if !output.Matches.MatchString(s) {
			t.Errorf(`Output in %v did not match regex "%v", got: %v`, name, output.Matches, s)
		}
	}
}
