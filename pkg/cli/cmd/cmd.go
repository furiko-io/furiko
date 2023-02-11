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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/utils/logging"
)

type RootCommand struct {
	streams *streams.Streams

	verbosity int
}

// NewRootCommand returns a new root command for the command-line utility.
func NewRootCommand(streams *streams.Streams) *cobra.Command {
	c := &RootCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:               common.CommandName,
		Short:             "Command-line utility to manage Furiko.",
		PersistentPreRunE: c.PersistentPreRunE,

		// Don't show help on error.
		SilenceUsage: true,
	}

	// Set IO streams.
	streams.SetCmdOutput(cmd)

	flags := cmd.PersistentFlags()
	flags.String("kubeconfig", "",
		"Path to the kubeconfig file to use for CLI requests.")
	flags.String("cluster", "",
		"The name of the kubeconfig cluster to use.")
	flags.String("context", "",
		"The name of the kubeconfig context to use.")
	flags.StringP("namespace", "n", "",
		"If present, the namespace scope for this CLI request.")
	flags.String("dynamic-config-name", "execution-dynamic-config",
		"Overrides the name of the dynamic cluster config.")
	flags.String("dynamic-config-namespace", "furiko-system",
		"Overrides the namespace of the dynamic cluster config.")
	flags.IntVarP(&c.verbosity, "v", "v", 0, "Sets the log level verbosity.")

	if err := completion.RegisterFlagCompletions(cmd, []completion.FlagCompletion{
		{FlagName: "namespace", Completer: &completion.ListNamespacesCompleter{}},
	}); err != nil {
		common.Fatal(err, common.DefaultErrorExitCode)
	}

	cmd.AddCommand(NewDisableCommand(streams))
	cmd.AddCommand(NewEnableCommand(streams))
	cmd.AddCommand(NewGetCommand(streams))
	cmd.AddCommand(NewKillCommand(streams))
	cmd.AddCommand(NewListCommand(streams))
	cmd.AddCommand(NewLogsCommand(streams))
	cmd.AddCommand(NewRunCommand(streams))

	return cmd
}

func (c *RootCommand) PersistentPreRunE(cmd *cobra.Command, args []string) error {
	if err := logging.SetLogLevel(klog.Level(c.verbosity)); err != nil {
		return errors.Wrap(err, "cannot set log level")
	}
	return nil
}
