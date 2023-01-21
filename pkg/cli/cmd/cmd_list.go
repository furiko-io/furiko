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

	"github.com/spf13/cobra"

	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/cli/streams"
)

func NewListCommand(streams *streams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all resources by kind.",
	}

	// Add common flags
	cmd.PersistentFlags().StringP("output", "o", string(printer.OutputFormatPretty),
		fmt.Sprintf("Output format. One of: %v", strings.Join(printer.GetAllOutputFormatStrings(), "|")))

	if err := RegisterFlagCompletions(cmd, []FlagCompletion{
		{FlagName: "output", CompletionFunc: (&CompletionHelper{}).FromSlice(printer.AllOutputFormats)},
	}); err != nil {
		Fatal(err, DefaultErrorExitCode)
	}

	cmd.AddCommand(NewListJobCommand(streams))
	cmd.AddCommand(NewListJobConfigCommand(streams))

	return cmd
}
