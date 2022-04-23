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

	"github.com/spf13/cobra"
)

func NewListCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List multiple resources.",
	}

	// Add common flags
	cmd.PersistentFlags().StringP("output", "o", "pretty",
		"Output format. One of: pretty|json|yaml|name")

	cmd.AddCommand(NewListJobCommand(ctx))
	cmd.AddCommand(NewListJobConfigCommand(ctx))

	return cmd
}
