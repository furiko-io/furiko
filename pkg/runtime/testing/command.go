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
	"bytes"
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
	fakeclock "k8s.io/utils/clock/testing"

	"github.com/furiko-io/furiko/pkg/cli/cmd"
	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/console"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	testTimeout = time.Second * 5
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

	// Sets the current time.
	Now time.Time

	// Input rules for standard input.
	Stdin Input

	// Output rules for standard output.
	Stdout Output

	// Output rules for standard error.
	Stderr Output

	// Procedure for running custom assertions.
	Procedure func(t *testing.T, rc RunContext)

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

	// If specified, expects the output to match all the given regular expressions.
	MatchesAll []*regexp.Regexp

	// If specified, expects that the output to NOT contain the given string.
	Excludes string

	// If specified, expects that the output to NOT contain any of the given strings.
	ExcludesAll []string

	// If non-nil, expects that the number of lines printed matches.
	NumLines *int
}

type RunContext struct {
	// Console to interact with the TTY.
	Console *console.Console

	// CtrlContext to interact with clientsets.
	CtrlContext *mock.Context

	// Context for the command. It will be canceled on interrupt.
	Context context.Context

	// Cancel the Context. Exposed to allow tests to interrupt the command midway.
	Cancel context.CancelFunc
}

func (c *CommandTest) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ctrlContext := mock.NewContext()
	common.SetCtrlContext(ctrlContext)
	iostreams, _, stdout, stderr := genericclioptions.NewTestIOStreams()

	// Execute the command and perform the procedure, before verifying the expected outputs.
	c.run(ctx, ctrlContext, t, iostreams)
	c.verify(t, ctrlContext, stdout, stderr)

	// The context was canceled by the time that we got here.
	// Return an error and show the output that was printed so far.
	if ctx.Err() != nil {
		t.Errorf("test did not complete after %v: %v", testTimeout, ctx.Err())
		t.Logf("command stdout =\n%s", stdout.String())
		t.Logf("command stderr =\n%s", stderr.String())
	}
}

func (c *CommandTest) run(ctx context.Context, ctrlContext *mock.Context, t *testing.T, iostreams genericclioptions.IOStreams) {
	// Override the shared context.
	client := ctrlContext.MockClientsets()
	assert.NoError(t, InitFixtures(ctx, client, c.Fixtures))
	client.ClearActions()

	// Set up clock.
	now := time.Now()
	if !c.Now.IsZero() {
		now = c.Now
	}
	ktime.Clock = fakeclock.NewFakeClock(now)

	// Run command with I/O.
	c.runCommand(ctx, t, ctrlContext, iostreams)
}

func (c *CommandTest) verify(t *testing.T, ctrlContext *mock.Context, stdout, stderr *bytes.Buffer) {
	client := ctrlContext.MockClientsets()

	// Ensure that output matches.
	c.checkOutput(t, "stdout", stdout.String(), c.Stdout)
	c.checkOutput(t, "stderr", stderr.String(), c.Stderr)

	// Compare actions.
	c.WantActions.Compare(t, client)
}

// runCommand will execute the command, setting up all I/O streams and blocking
// until the streams are done.
//
// Reference:
// https://github.com/AlecAivazis/survey/blob/93657ef69381dd1ffc7a4a9cfe5a2aefff4ca4ad/survey_posix_test.go#L15
func (c *CommandTest) runCommand(ctx context.Context, t *testing.T, ctrlContext *mock.Context, iostreams genericclioptions.IOStreams) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	console, err := console.NewConsole(t, iostreams.Out)
	if err != nil {
		t.Fatalf("failed to create console: %v", err)
	}
	defer console.Close()

	// The procedure will be executed in the background.
	// Once completed, the returned channel will be closed.
	var done <-chan struct{}
	if c.Procedure != nil {
		done = c.runProcedure(t, c.Procedure, RunContext{
			Console:     console,
			Context:     ctx,
			Cancel:      cancel,
			CtrlContext: ctrlContext,
		})
	} else {
		done = console.Run(c.Stdin.Procedure)
	}

	// Execute the command.
	command := cmd.NewRootCommand(streams.NewTTYStreams(console.Tty()))
	command.SetArgs(c.Args)
	testutils.WantError(t, c.WantError, command.ExecuteContext(ctx), fmt.Sprintf("Run error with args: %v", c.Args))

	// Wait until the TTY completely prints all output, before closing the PTY to
	// unblock any ExpectEOF() calls.
	// TODO: No better way other than to sleep?
	time.Sleep(time.Millisecond * 50)
	assert.NoError(t, console.Tty().Close())

	// Block until procedure terminates fully.
	<-done
}

// runProcedure starts the procedure in the background, and returns a channel
// that will be closed once the PTY output is closed.
func (c *CommandTest) runProcedure(t *testing.T, procedure func(t *testing.T, rc RunContext), rc RunContext) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)

		// Run the procedure.
		procedure(t, rc)

		// Waits for the TTY to be closed.
		_, err := rc.Console.ExpectEOF()
		assert.NoError(t, err, "Failed to ExpectEOF")
	}()
	return done
}

func (c *CommandTest) checkOutput(t *testing.T, name, s string, output Output) { // nolint:gocognit
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

	if output.Excludes != "" && strings.Contains(s, output.Excludes) {
		t.Errorf(`Output in %v contains unexpected string "%v", got: %v`, name, output.Excludes, s)
	}

	if len(output.ExcludesAll) > 0 {
		for _, excludes := range output.ExcludesAll {
			if strings.Contains(s, excludes) {
				t.Errorf(`Output in %v contains unexpected string "%v", got: %v`, name, excludes, s)
			}
		}
	}

	if output.Matches != nil {
		if !output.Matches.MatchString(s) {
			t.Errorf(`Output in %v did not match regex "%v", got: %v`, name, output.Matches, s)
		}
	}

	if len(output.MatchesAll) > 0 {
		for _, regex := range output.MatchesAll {
			if !regex.MatchString(s) {
				t.Errorf(`Output in %v did not match regex "%v", got: %v`, name, regex, s)
			}
		}
	}

	if output.NumLines != nil {
		numLines := strings.Count(s, "\n")
		if len(s) > 0 && s[len(s)-1] != '\n' {
			numLines++
		}
		if *output.NumLines != numLines {
			t.Errorf("Output in %v did not match expected number of lines, wanted %v, got %v: %v", name, *output.NumLines, numLines, s)
		}
	}
}
