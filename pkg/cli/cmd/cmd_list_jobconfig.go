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
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/cli/streams"
)

var (
	ListJobConfigExample = PrepareExample(`
# List all JobConfigs in current namespace.
{{.CommandName}} list jobconfig

# List all JobConfigs in JSON format.
{{.CommandName}} list jobconfig -o json`)
)

type ListJobConfigCommand struct {
	streams *streams.Streams
}

func NewListJobConfigCommand(streams *streams.Streams) *cobra.Command {
	c := &ListJobConfigCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:     "jobconfig",
		Aliases: []string{"jobconfigs"},
		Short:   "Displays information about multiple JobConfigs.",
		Example: ListJobConfigExample,
		PreRunE: PrerunWithKubeconfig,
		Args:    cobra.ExactArgs(0),
		RunE:    c.Run,
	}

	return cmd
}

func (c *ListJobConfigCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
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

	if len(jobConfigList.Items) == 0 {
		c.streams.Printf("No job configs found in %v namespace.\n", namespace)
		return nil
	}

	return c.PrintJobConfigs(output, jobConfigList)
}

func (c *ListJobConfigCommand) PrintJobConfigs(
	output printer.OutputFormat,
	jobConfigList *execution.JobConfigList,
) error {
	// Handle pretty print as a special case.
	if output == printer.OutputFormatPretty {
		c.prettyPrint(jobConfigList.Items)
		return nil
	}

	// Extract list.
	items := make([]printer.Object, 0, len(jobConfigList.Items))
	for _, job := range jobConfigList.Items {
		job := job
		items = append(items, &job)
	}

	return printer.PrintObjects(execution.GVKJobConfig, output, c.streams.Out, items)
}

func (c *ListJobConfigCommand) prettyPrint(jobConfigs []execution.JobConfig) {
	p := printer.NewTablePrinter(c.streams.Out)
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
		lastScheduled = formatter.FormatTimeAgo(jobConfig.Status.LastScheduled)
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
