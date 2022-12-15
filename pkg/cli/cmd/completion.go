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

package cmd

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// CompletionFunc knows how to return completions.
type CompletionFunc func(ctx context.Context, ctrlContext controllercontext.Context, cmd *cobra.Command) ([]string, error)

// FlagCompletion defines a single flag completion entry.
type FlagCompletion struct {
	FlagName       string
	CompletionFunc CompletionFunc
}

// RegisterFlagCompletions registers all FlagCompletion entries and returns an error if any flag returned an error.
func RegisterFlagCompletions(cmd *cobra.Command, completions []FlagCompletion) error {
	for _, completion := range completions {
		if err := cmd.RegisterFlagCompletionFunc(completion.FlagName, MakeCobraCompletionFunc(completion.CompletionFunc)); err != nil {
			return errors.Wrapf(err, `cannot register flag completion func for "%v"`, completion.FlagName)
		}
	}
	return nil
}

// MakeCobraCompletionFunc converts a CompletionFunc into a cobra completion function.
func MakeCobraCompletionFunc(f CompletionFunc) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx := cmd.Context()
		if err := SetupCtrlContext(cmd); err != nil {
			Fatal(errors.Wrap(err, "cannot set up context"), DefaultErrorExitCode)
		}
		completions, err := f(ctx, ctrlContext, cmd)
		if err != nil {
			Fatal(err, DefaultErrorExitCode)
			return nil, cobra.ShellCompDirectiveError
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}
