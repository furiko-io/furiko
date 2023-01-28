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
	"k8s.io/apimachinery/pkg/runtime"

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
	streams   *streams.Streams
	output    printer.OutputFormat
	noHeaders bool
	watch     bool
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
		RunE: RunAllE(
			c.Complete,
			c.Run,
		),
	}

	return cmd
}

func (c *ListJobConfigCommand) Complete(cmd *cobra.Command, args []string) error {
	c.output = GetOutputFormat(cmd)
	c.noHeaders = GetFlagBool(cmd, "no-headers")
	c.watch = GetFlagBool(cmd, "watch")
	return nil
}

func (c *ListJobConfigCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := GetNamespace(cmd)
	if err != nil {
		return err
	}

	var options metav1.ListOptions
	jobConfigList, err := client.JobConfigs(namespace).List(ctx, options)
	if err != nil {
		return errors.Wrapf(err, "cannot list job configs")
	}

	if len(jobConfigList.Items) == 0 && !c.watch {
		c.streams.Printf("No job configs found in %v namespace.\n", namespace)
		return nil
	}

	p := printer.NewTablePrinter(c.streams.Out)

	if len(jobConfigList.Items) > 0 {
		if err := c.PrintJobConfigs(p, jobConfigList); err != nil {
			return errors.Wrapf(err, "cannot print jobconfigs")
		}
	}

	if c.watch {
		watchOptions := *options.DeepCopy()
		watchOptions.ResourceVersion = jobConfigList.ResourceVersion
		watch, err := client.JobConfigs(namespace).Watch(ctx, watchOptions)
		if err != nil {
			return errors.Wrapf(err, "cannot watch jobconfigs")
		}

		return WatchAndPrint(ctx, watch, func(obj runtime.Object) (bool, error) {
			if jobConfig, ok := obj.(*execution.JobConfig); ok {
				return true, c.PrintJobConfig(p, jobConfig)
			}
			return false, nil
		})
	}

	return nil
}

func (c *ListJobConfigCommand) PrintJobConfigs(p *printer.TablePrinter, jobConfigList *execution.JobConfigList) error {
	// Handle pretty print as a special case.
	if c.output == printer.OutputFormatPretty {
		c.prettyPrint(p, jobConfigList.Items)
		return nil
	}

	// Extract list.
	items := make([]printer.Object, 0, len(jobConfigList.Items))
	for _, job := range jobConfigList.Items {
		job := job
		items = append(items, &job)
	}

	return printer.PrintObjects(execution.GVKJobConfig, c.output, c.streams.Out, items)
}

func (c *ListJobConfigCommand) PrintJobConfig(p *printer.TablePrinter, jobConfig *execution.JobConfig) error {
	// Handle pretty print as a special case.
	if c.output == printer.OutputFormatPretty {
		jobConfigList := []execution.JobConfig{*jobConfig}
		c.prettyPrint(p, jobConfigList)
		return nil
	}

	return printer.PrintObject(execution.GVKJobConfig, c.output, c.streams.Out, jobConfig)
}

func (c *ListJobConfigCommand) prettyPrint(p *printer.TablePrinter, jobConfigs []execution.JobConfig) {
	var headers []string
	if !c.noHeaders {
		headers = c.makeJobHeader()
	}
	p.Print(headers, c.makeJobRows(jobConfigs))
}

func (c *ListJobConfigCommand) makeJobHeader() []string {
	return []string{
		"NAME",
		"STATE",
		"ACTIVE",
		"QUEUED",
		"LAST EXECUTED",
		"LAST SCHEDULED",
		"CRON SCHEDULE",
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
	lastExecuted := ""
	lastScheduled := ""

	if schedule := jobConfig.Spec.Schedule; schedule != nil && schedule.Cron != nil {
		cronSchedule = schedule.Cron.Expression
	}
	if !jobConfig.Status.LastExecuted.IsZero() {
		lastExecuted = formatter.FormatTimeAgo(jobConfig.Status.LastExecuted)
	}
	if !jobConfig.Status.LastScheduled.IsZero() {
		lastScheduled = formatter.FormatTimeAgo(jobConfig.Status.LastScheduled)
	}

	return []string{
		jobConfig.Name,
		string(jobConfig.Status.State),
		strconv.Itoa(int(jobConfig.Status.Active)),
		strconv.Itoa(int(jobConfig.Status.Queued)),
		lastExecuted,
		lastScheduled,
		cronSchedule,
	}
}
