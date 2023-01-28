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

package completion

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// Func is a completion func, that knows how to return completions.
type Func func(ctx context.Context, ctrlContext controllercontext.Context, cmd *cobra.Command) ([]string, error)

// Completer is the interface of Func.
type Completer interface {
	Complete(ctx context.Context, ctrlContext controllercontext.Context, cmd *cobra.Command) ([]string, error)
}

// FlagCompletion defines a single flag completion entry.
type FlagCompletion struct {
	FlagName       string
	CompletionFunc Func
	Completer      Completer
}

type cobraCompletionFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// RegisterFlagCompletions registers all FlagCompletion entries and returns an error if any flag returned an error.
func RegisterFlagCompletions(cmd *cobra.Command, completions []FlagCompletion) error {
	for _, completion := range completions {
		if completion.FlagName == "" {
			return fmt.Errorf("flag name must be specified: %v", completion)
		}

		var completionFunc cobraCompletionFunc
		if completion.CompletionFunc != nil {
			completionFunc = MakeCobraCompletionFunc(completion.CompletionFunc)
		} else if completion.Completer != nil {
			completionFunc = CompleterToCobraCompletionFunc(completion.Completer)
		}
		if completionFunc == nil {
			return fmt.Errorf(`no completion func specified for "%v"`, completion.FlagName)
		}

		if err := cmd.RegisterFlagCompletionFunc(completion.FlagName, completionFunc); err != nil {
			return errors.Wrapf(err, `cannot register flag completion func for "%v"`, completion.FlagName)
		}
	}
	return nil
}

// MakeCobraCompletionFunc converts a Func into a cobra completion function.
func MakeCobraCompletionFunc(f Func) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx := cmd.Context()
		if err := common.SetupCtrlContext(cmd); err != nil {
			common.Fatal(errors.Wrap(err, "cannot set up context"), common.DefaultErrorExitCode)
		}
		completions, err := f(ctx, common.GetCtrlContext(), cmd)
		if err != nil {
			common.Fatal(err, common.DefaultErrorExitCode)
			return nil, cobra.ShellCompDirectiveError
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// CompleterToCobraCompletionFunc converts a Completer into a cobra completion function.
func CompleterToCobraCompletionFunc(c Completer) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return MakeCobraCompletionFunc(c.Complete)
}
