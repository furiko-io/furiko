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
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	"github.com/furiko-io/furiko/pkg/cli/format"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/execution/util/cron"
)

var (
	GetJobConfigExample = common.PrepareExample(`
# Get a single JobConfig in the current namespace.
{{.CommandName}} get jobconfig daily-send-email

# Get multiple JobConfigs by name in the current namespace.
{{.CommandName}} get jobconfig daily-send-email weekly-report-email

# Get a single JobConfig in JSON format.
{{.CommandName}} get jobconfig daily-send-email -o json`)
)

type GetJobConfigCommand struct {
	streams *streams.Streams

	output printer.OutputFormat

	// Cached cron parser.
	cronParser *cron.Parser
}

func NewGetJobConfigCommand(streams *streams.Streams) *cobra.Command {
	c := &GetJobConfigCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:               "jobconfig",
		Aliases:           []string{"jobconfigs"},
		Short:             "Displays information about one or more JobConfigs.",
		Example:           GetJobConfigExample,
		PreRunE:           common.PrerunWithKubeconfig,
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completion.CompleterToCobraCompletionFunc(&completion.ListJobConfigsCompleter{}),
		RunE: common.RunAllE(
			c.Complete,
			c.Run,
		),
	}

	return cmd
}

func (c *GetJobConfigCommand) Complete(cmd *cobra.Command, args []string) error {
	c.output = common.GetOutputFormat(cmd)

	// Prepare parser.
	c.cronParser = cron.NewParserFromConfig(common.GetCronDynamicConfig(cmd))

	return nil
}

func (c *GetJobConfigCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client := common.GetCtrlContext().Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := common.GetNamespace(cmd)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return errors.New("at least one name is required")
	}

	jobConfigs := make([]*execution.JobConfig, 0, len(args))
	for _, name := range args {
		jobConfig, err := client.JobConfigs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "cannot get job config")
		}
		jobConfigs = append(jobConfigs, jobConfig)
	}

	return c.PrintJobConfigs(c.output, jobConfigs)
}

func (c *GetJobConfigCommand) PrintJobConfigs(output printer.OutputFormat, jobConfigs []*execution.JobConfig) error {
	// Handle pretty print as a special case.
	if output == printer.OutputFormatPretty {
		p := printer.NewDescriptionPrinter(c.streams.Out)
		for i, jobConfig := range jobConfigs {
			if err := c.prettyPrintJobConfig(jobConfig); err != nil {
				return err
			}
			if i+1 < len(jobConfigs) {
				p.Divider()
			}
		}
		return nil
	}

	// Print items.
	items := make([]printer.Object, 0, len(jobConfigs))
	for _, job := range jobConfigs {
		items = append(items, job)
	}

	return printer.PrintObjects(execution.GVKJobConfig, output, c.streams.Out, items)
}

func (c *GetJobConfigCommand) prettyPrintJobConfig(jobConfig *execution.JobConfig) error {
	p := printer.NewDescriptionPrinter(c.streams.Out)

	// Print the metadata.
	p.Header("Job Config Metadata")
	p.Descriptions(c.prettyPrintJobConfigMetadata(jobConfig))

	// Print each section with a header if it is non-empty.
	sections := []printer.SectionGenerator{
		printer.ToSectionGenerator(printer.Section{Name: "Concurrency", Descs: c.prettyPrintJobConfigConcurrency(jobConfig)}),
		func() (printer.Section, error) {
			descs, err := c.prettyPrintJobConfigSchedule(jobConfig)
			if err != nil {
				return printer.Section{}, err
			}
			return printer.Section{Name: "Schedule", Descs: descs}, nil
		},
		printer.ToSectionGenerator(printer.Section{Name: "Job Status", Descs: c.prettyPrintJobConfigStatus(jobConfig)}),
	}

	for _, gen := range sections {
		section, err := gen()
		if err != nil {
			return err
		}
		if len(section.Descs) > 0 {
			p.NewLine()
			p.Header(section.Name)
			p.Descriptions(section.Descs)
		}
	}
	return nil
}

func (c *GetJobConfigCommand) prettyPrintJobConfigMetadata(jobConfig *execution.JobConfig) [][]string {
	return [][]string{
		{"Name", jobConfig.Name},
		{"Namespace", jobConfig.Namespace},
		{"Created", format.TimeWithTimeAgo(&jobConfig.CreationTimestamp)},
	}
}

func (c *GetJobConfigCommand) prettyPrintJobConfigConcurrency(jobConfig *execution.JobConfig) [][]string {
	return [][]string{
		{"Policy", string(jobConfig.Spec.Concurrency.Policy)},
	}
}

func (c *GetJobConfigCommand) prettyPrintJobConfigSchedule(jobConfig *execution.JobConfig) ([][]string, error) {
	schedule := jobConfig.Spec.Schedule
	if schedule == nil {
		return nil, nil
	}

	result := [][]string{
		{"Enabled", strconv.FormatBool(!schedule.Disabled)},
	}

	if cronSchedule := schedule.Cron; cronSchedule != nil {
		result = append(result, []string{"Cron Expression(s)", strings.Join(schedule.Cron.GetExpressions(), "\n")})
		if cronSchedule.Timezone != "" {
			result = append(result, []string{"Cron Timezone", cronSchedule.Timezone})
		}
	}

	if constraints := schedule.Constraints; constraints != nil {
		result = printer.MaybeAppendTimeAgo(result, "Not Before", constraints.NotBefore)
		result = printer.MaybeAppendTimeAgo(result, "Not After", constraints.NotAfter)
	}

	// Here we evaluate the next schedule as a convenience to the user. Note that is
	// very likely just an approximation, since many factors influence the actual
	// time that the next job is scheduled.
	if !schedule.Disabled && schedule.Cron != nil && len(schedule.Cron.GetExpressions()) > 0 {
		resp, err := c.prettyPrintNextSchedule(jobConfig)
		if err != nil {
			return nil, err
		}
		result = append(result, resp...)
	}

	return result, nil
}

func (c *GetJobConfigCommand) prettyPrintNextSchedule(jobConfig *execution.JobConfig) ([][]string, error) {
	next, err := FormatNextSchedule(jobConfig, c.cronParser, format.TimeAgo)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot format next schedule time")
	}
	return [][]string{{"Next Schedule", next}}, nil
}

func (c *GetJobConfigCommand) prettyPrintJobConfigStatus(jobConfig *execution.JobConfig) [][]string {
	result := [][]string{
		{"State", string(jobConfig.Status.State)},
		{"Queued Jobs", strconv.Itoa(int(jobConfig.Status.Queued))},
		{"Active Jobs", strconv.Itoa(int(jobConfig.Status.Active))},
	}

	lastScheduled := "Never"
	lastExecuted := "Never"
	if t := jobConfig.Status.LastScheduled; !t.IsZero() {
		lastScheduled = format.TimeWithTimeAgo(t)
	}
	if t := jobConfig.Status.LastExecuted; !t.IsZero() {
		lastExecuted = format.TimeWithTimeAgo(t)
	}
	result = append(result, []string{"Last Scheduled", lastScheduled})
	result = append(result, []string{"Last Executed", lastExecuted})

	return result
}
