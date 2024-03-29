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
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/format"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/execution/util/cron"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
	"github.com/furiko-io/furiko/pkg/utils/sets"
)

var (
	ListJobConfigExample = common.PrepareExample(`
# List all JobConfigs in current namespace.
{{.CommandName}} list jobconfig

# List all JobConfigs across all namespaces.
{{.CommandName}} list jobconfig -A

# List only JobConfigs with a schedule.
{{.CommandName}} list jobconfig --scheduled

# List adhoc-only JobConfigs (i.e. does not have a schedule).
{{.CommandName}} list jobconfig --adhoc-only

# List JobConfigs in JSON format.
{{.CommandName}} list jobconfig -o json`)
)

type ListJobConfigCommand struct {
	*baseListCommand

	streams         *streams.Streams
	output          printer.OutputFormat
	noHeaders       bool
	scheduled       bool
	adhocOnly       bool
	scheduleEnabled bool
	watch           bool
	allNamespaces   bool

	// Cached cron parser.
	cronParser *cron.Parser

	// Cached set of job UIDs that were previously filtered in.
	// If it was displayed before, we don't want to filter it out afterwards.
	// This is mostly useful for --watch.
	cachedFiltered sets.String
}

func NewListJobConfigCommand(streams *streams.Streams) *cobra.Command {
	c := &ListJobConfigCommand{
		baseListCommand: newBaseListCommand(),
		streams:         streams,
		cachedFiltered:  sets.NewString(),
	}

	cmd := &cobra.Command{
		Use:     "jobconfig",
		Aliases: []string{"jobconfigs"},
		Short:   "Displays information about multiple JobConfigs.",
		Example: ListJobConfigExample,
		PreRunE: common.PrerunWithKubeconfig,
		Args:    cobra.ExactArgs(0),
		RunE: common.RunAllE(
			c.Complete,
			c.Validate,
			c.Run,
		),
	}

	cmd.Flags().BoolVar(&c.scheduled, "scheduled", false, "Only return job configs with a schedule.")
	cmd.Flags().BoolVar(&c.scheduleEnabled, "schedule-enabled", false, "Only return job configs with a schedule and are enabled.")
	cmd.Flags().BoolVar(&c.adhocOnly, "adhoc-only", false, "Only return job configs without a schedule (i.e. adhoc-only job configs).")

	return cmd
}

func (c *ListJobConfigCommand) Complete(cmd *cobra.Command, args []string) error {
	if err := c.baseListCommand.Complete(cmd, args); err != nil {
		return err
	}

	c.output = common.GetOutputFormat(cmd)
	c.noHeaders = common.GetFlagBool(cmd, "no-headers")
	c.watch = common.GetFlagBool(cmd, "watch")
	c.allNamespaces = common.GetFlagBool(cmd, "all-namespaces")

	// Prepare parser.
	c.cronParser = cron.NewParserFromConfig(common.GetCronDynamicConfig(cmd))

	return nil
}

func (c *ListJobConfigCommand) Validate(cmd *cobra.Command, args []string) error {
	if c.scheduled && c.adhocOnly {
		return fmt.Errorf("cannot specify --scheduled and --adhoc-only together")
	}
	if c.scheduleEnabled && c.adhocOnly {
		return fmt.Errorf("cannot specify --schedule-enabled and --adhoc-only together")
	}

	return nil
}

func (c *ListJobConfigCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client := common.GetCtrlContext().Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := common.GetNamespace(cmd)
	if err != nil {
		return err
	}

	var options metav1.ListOptions

	// Add selectors.
	if c.labelSelector != nil {
		options.LabelSelector = c.labelSelector.String()
	}
	if c.fieldSelector != nil {
		options.FieldSelector = c.fieldSelector.String()
	}

	jobConfigList, err := client.JobConfigs(namespace).List(ctx, options)
	if err != nil {
		return errors.Wrapf(err, "cannot list job configs")
	}

	items := c.FilterJobConfigs(jobConfigList.Items)

	if len(items) == 0 && !c.watch {
		c.streams.Printf("No job configs found in %v namespace.\n", namespace)
		return nil
	}

	p := printer.NewTablePrinter(c.streams.Out)

	if len(items) > 0 {
		if err := c.PrintJobConfigs(p, items); err != nil {
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

		return WatchAndPrint(ctx, watch, nil, func(jobConfig *execution.JobConfig) error {
			return c.PrintJobConfig(p, jobConfig)
		})
	}

	return nil
}

func (c *ListJobConfigCommand) PrintJobConfigs(p *printer.TablePrinter, jobConfigs []*execution.JobConfig) error {
	// Handle pretty print as a special case.
	if c.output == printer.OutputFormatPretty {
		c.prettyPrint(p, jobConfigs)
		return nil
	}

	// Extract list.
	items := make([]printer.Object, 0, len(jobConfigs))
	for _, job := range jobConfigs {
		items = append(items, job)
	}

	return printer.PrintObjects(execution.GVKJobConfig, c.output, c.streams.Out, items)
}

func (c *ListJobConfigCommand) PrintJobConfig(p *printer.TablePrinter, jobConfig *execution.JobConfig) error {
	// Handle pretty print as a special case.
	if c.output == printer.OutputFormatPretty {
		c.prettyPrint(p, []*execution.JobConfig{jobConfig})
		return nil
	}

	return printer.PrintObject(execution.GVKJobConfig, c.output, c.streams.Out, jobConfig)
}

func (c *ListJobConfigCommand) prettyPrint(p *printer.TablePrinter, jobConfigs []*execution.JobConfig) {
	var headers []string
	if !c.noHeaders {
		headers = c.makeJobHeader()
	}
	p.Print(headers, c.makeJobRows(jobConfigs))
}

func (c *ListJobConfigCommand) makeJobHeader() []string {
	var columns []string
	if c.allNamespaces {
		columns = append(columns, "NAMESPACE")
	}
	columns = append(columns, []string{
		"NAME",
		"STATE",
		"ACTIVE",
		"QUEUED",
		"LAST EXECUTED",
		"LAST SCHEDULED",
		"CRON SCHEDULE",
		"NEXT SCHEDULE",
	}...)
	return columns
}

func (c *ListJobConfigCommand) makeJobRows(jobConfigs []*execution.JobConfig) [][]string {
	rows := make([][]string, 0, len(jobConfigs))
	for _, jobConfig := range jobConfigs {
		rows = append(rows, c.makeJobRow(jobConfig))
	}
	return rows
}

func (c *ListJobConfigCommand) makeJobRow(jobConfig *execution.JobConfig) []string {
	var cronSchedule, nextSchedule string
	if schedule := jobConfig.Spec.Schedule; schedule != nil && schedule.Cron != nil {
		cronSchedule = schedule.Cron.GetExpressions().String()
		if next, err := format.NextScheduleForJobConfig(jobConfig, c.cronParser, format.TimeAgo); err == nil {
			nextSchedule = next
		}
	}

	var row []string
	if c.allNamespaces {
		row = append(row, jobConfig.Namespace)
	}
	row = append(row, []string{
		jobConfig.Name,
		string(jobConfig.Status.State),
		strconv.Itoa(int(jobConfig.Status.Active)),
		strconv.Itoa(int(jobConfig.Status.Queued)),
		format.TimeAgo(jobConfig.Status.LastExecuted),
		format.TimeAgo(jobConfig.Status.LastScheduled),
		cronSchedule,
		nextSchedule,
	}...)
	return row
}

// FilterJobConfigs returns a filtered list of job configs.
func (c *ListJobConfigCommand) FilterJobConfigs(jobConfigs []execution.JobConfig) []*execution.JobConfig {
	filtered := make([]*execution.JobConfig, 0, len(jobConfigs))
	for _, jobConfig := range jobConfigs {
		jobConfig := jobConfig.DeepCopy()
		if c.FilterJobConfig(jobConfig) {
			filtered = append(filtered, jobConfig)
		}
	}
	return filtered
}

// FilterJobConfig returns true if the job config is retained in the filter.
func (c *ListJobConfigCommand) FilterJobConfig(jobConfig *execution.JobConfig) bool {
	// Check if filtered in, and store in the filter cache.
	if c.filterJobConfig(jobConfig) {
		c.cachedFiltered.Insert(string(jobConfig.UID))
	}

	// Look up exclusively in the filter cache.
	// If a job config was previously filtered in, we will always allow it to pass in subsequent calls.
	return c.cachedFiltered.Has(string(jobConfig.UID))
}

func (c *ListJobConfigCommand) filterJobConfig(jobConfig *execution.JobConfig) bool {
	// Filter out job configs which are not scheduled.
	if c.scheduled && jobConfig.Spec.Schedule == nil {
		return false
	}

	// Filter out job configs which are scheduled.
	if c.adhocOnly && jobConfig.Spec.Schedule != nil {
		return false
	}

	// Filter out job configs which are not scheduled or are disabled.
	if c.scheduleEnabled && !jobconfig.IsScheduleEnabled(jobConfig) {
		return false
	}

	return true
}
