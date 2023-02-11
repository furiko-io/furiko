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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/cli/streams"
)

type ListCommand struct {
	streams *streams.Streams
}

func NewListCommand(streams *streams.Streams) *cobra.Command {
	c := &ListCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all resources by kind.",
	}

	c.AddSubcommand(cmd, NewListJobCommand(streams))
	c.AddSubcommand(cmd, NewListJobConfigCommand(streams))

	return cmd
}

// AddSubcommand will register subcommand flags and add it to the parent command.
func (c *ListCommand) AddSubcommand(cmd, subCommand *cobra.Command) {
	c.RegisterFlags(subCommand)
	cmd.AddCommand(subCommand)
}

// RegisterFlags will register flags for the given subcommand.
func (c *ListCommand) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("output", "o", string(printer.OutputFormatPretty),
		fmt.Sprintf("Output format. One of: %v", strings.Join(printer.GetAllOutputFormatStrings(), "|")))
	cmd.Flags().Bool("no-headers", false,
		"When using the default or custom-column output format, don't print headers (default print headers).")
	cmd.Flags().BoolP("watch", "w", false,
		"After listing the requested object(s), watch for changes and print them to the standard output.")
	cmd.Flags().StringP("selector", "l", "",
		"Selector (label query) to filter on, supports '=', '==', and '!=' (e.g. -l key1=value1,key2=value2). "+
			"Matching objects must satisfy all of the specified label constraints.")
	cmd.Flags().String("field-selector", "",
		"Selector (field query) to filter on, supports '=', '==', and '!=' (e.g. --field-selector key1=value1,key2=value2). "+
			"The server only supports a limited number of field queries per type.")
	cmd.Flags().BoolP("all-namespaces", "A", false,
		"If specified, list the requested objects across all namespaces. Namespace in current context is ignored even if specified with --namespace.")

	if err := completion.RegisterFlagCompletions(cmd, []completion.FlagCompletion{
		{FlagName: "output", Completer: completion.NewSliceCompleter(printer.AllOutputFormats)},
	}); err != nil {
		common.Fatal(err, common.DefaultErrorExitCode)
	}
}

type baseListCommand struct {
	labelSelector labels.Selector
	fieldSelector labels.Selector
}

func newBaseListCommand() *baseListCommand {
	return &baseListCommand{
		labelSelector: labels.NewSelector(),
		fieldSelector: labels.NewSelector(),
	}
}

func (c *baseListCommand) Complete(cmd *cobra.Command, args []string) error {
	// Parse --label-selector.
	if labelSelector := common.GetFlagString(cmd, "selector"); labelSelector != "" {
		selector, err := labels.Parse(labelSelector)
		if err != nil {
			return errors.Wrapf(err, "invalid value for --selector: cannot parse %v as selector", labelSelector)
		}
		c.labelSelector = selector
	}

	// Parse --field-selector.
	if fieldSelector := common.GetFlagString(cmd, "field-selector"); fieldSelector != "" {
		selector, err := labels.Parse(fieldSelector)
		if err != nil {
			return errors.Wrapf(err, "invalid value for --field-selector: cannot parse %v as selector", fieldSelector)
		}
		c.fieldSelector = selector
	}

	return nil
}
