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
	"k8s.io/klog/v2"

	"github.com/furiko-io/furiko/pkg/utils/logging"
)

type RootCommand struct {
	kubeconfig         string
	namespace          string
	verbosity          int
	dynConfigName      string
	dynConfigNamespace string
}

// NewRootCommand returns a new root command for the command-line utility.
func NewRootCommand(ctx context.Context) *cobra.Command {
	c := &RootCommand{}
	cmd := &cobra.Command{
		Use:               "furiko",
		Short:             "Command-line utility to manage Furiko.",
		PersistentPreRunE: ToRunE(ctx, c),

		// Don't show help on error.
		SilenceUsage: true,
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&c.kubeconfig, "kubeconfig", "",
		"Path to the kubeconfig file to use for CLI requests.")
	flags.StringVarP(&c.namespace, "namespace", "n", "",
		"If present, the namespace scope for this CLI request.")
	flags.StringVar(&c.dynConfigName, "dynamic-config-name", "execution-dynamic-config",
		"Overrides the name of the dynamic cluster config.")
	flags.StringVar(&c.dynConfigNamespace, "dynamic-config-namespace", "furiko-system",
		"Overrides the namespace of the dynamic cluster config.")
	flags.IntVarP(&c.verbosity, "v", "v", 0, "Sets the log level verbosity.")

	cmd.AddCommand(NewGetCommand(ctx))
	cmd.AddCommand(NewListCommand(ctx))
	cmd.AddCommand(NewRunCommand(ctx))

	return cmd
}

func (c *RootCommand) Run(ctx context.Context, cmd *cobra.Command, args []string) error {
	if err := logging.SetLogLevel(klog.Level(c.verbosity)); err != nil {
		return errors.Wrap(err, "cannot set log level")
	}
	return nil
}
