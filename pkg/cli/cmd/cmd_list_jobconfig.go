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
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

type ListJobConfigCommand struct{}

func NewListJobConfigCommand(ctx context.Context) *cobra.Command {
	c := &ListJobConfigCommand{}
	cmd := &cobra.Command{
		Use:     "jobconfig",
		Aliases: []string{"jobconfigs"},
		Short:   "Displays information about multiple JobConfigs.",
		Example: `  # List all JobConfigs in current namespace.
  furiko list jobconfig

  # List all JobConfigs in JSON format.
  furiko list jobconfig -o json`,
		PreRunE: PrerunWithKubeconfig,
		Args:    cobra.ExactArgs(0),
		RunE:    ToRunE(ctx, c),
	}

	return cmd
}

func (c *ListJobConfigCommand) Run(ctx context.Context, cmd *cobra.Command, args []string) error {
	client := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := GetNamespace(cmd)
	if err != nil {
		return err
	}
	output, err := GetOutputFormat(cmd)
	if err != nil {
		return err
	}

	var options metav1.ListOptions
	jobConfigList, err := client.JobConfigs(namespace).List(ctx, options)
	if err != nil {
		return errors.Wrapf(err, "cannot list job configs")
	}
	jobConfigList.SetGroupVersionKind(execution.GVKJobConfigList)

	if len(jobConfigList.Items) == 0 {
		fmt.Printf("No job configs found in %v namespace.\n", namespace)
		return nil
	}

	return c.PrintJobConfigs(output, jobConfigList)
}

func (c *ListJobConfigCommand) PrintJobConfigs(output OutputFormat, jobConfigList *execution.JobConfigList) error {
	// Handle pretty print as a special case.
	if output == OutputFormatPretty {
		c.prettyPrint(jobConfigList.Items)
		return nil
	}

	return PrintObject(output, os.Stdout, jobConfigList)
}

func (c *ListJobConfigCommand) prettyPrint(jobConfigs []execution.JobConfig) {
	p := NewStdoutTablePrinter()
	p.Print(c.makeJobHeader(), c.makeJobRows(jobConfigs))
}

func (c *ListJobConfigCommand) makeJobHeader() []string {
	return []string{
		"NAME",
		"STATE",
		"ACTIVE",
		"QUEUED",
		"CRON SCHEDULE",
		"LAST SCHEDULED",
	}
}

func (c *ListJobConfigCommand) makeJobRows(jobConfigs []execution.JobConfig) [][]string {
	rows := make([][]string, 0, len(jobConfigs))
	for _, jobConfig := range jobConfigs {
		jobConfig := jobConfig
		rows = append(rows, c.makeJobRow(&jobConfig))
	}
	return rows
}

func (c *ListJobConfigCommand) makeJobRow(jobConfig *execution.JobConfig) []string {
	cronSchedule := ""
	lastScheduled := ""

	if schedule := jobConfig.Spec.Schedule; schedule != nil && schedule.Cron != nil {
		cronSchedule = schedule.Cron.Expression
	}
	if !jobConfig.Status.LastScheduled.IsZero() {
		lastScheduled = FormatTimeAgo(jobConfig.Status.LastScheduled)
	}

	return []string{
		jobConfig.Name,
		string(jobConfig.Status.State),
		strconv.Itoa(int(jobConfig.Status.Active)),
		strconv.Itoa(int(jobConfig.Status.Queued)),
		cronSchedule,
		lastScheduled,
	}
}
