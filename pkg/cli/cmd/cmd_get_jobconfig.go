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
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/config"
	"github.com/furiko-io/furiko/pkg/execution/util/cronparser"
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

	return c.PrintJobConfigs(ctx, cmd, c.output, jobConfigs)
}

func (c *GetJobConfigCommand) PrintJobConfigs(
	ctx context.Context,
	cmd *cobra.Command,
	output printer.OutputFormat,
	jobConfigs []*execution.JobConfig,
) error {
	// Handle pretty print as a special case.
	if output == printer.OutputFormatPretty {
		p := printer.NewDescriptionPrinter(c.streams.Out)
		for i, jobConfig := range jobConfigs {
			if err := c.prettyPrintJobConfig(ctx, cmd, jobConfig); err != nil {
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

func (c *GetJobConfigCommand) prettyPrintJobConfig(
	ctx context.Context,
	cmd *cobra.Command,
	jobConfig *execution.JobConfig,
) error {
	p := printer.NewDescriptionPrinter(c.streams.Out)

	// Print the metadata.
	p.Header("Job Config Metadata")
	p.Descriptions(c.prettyPrintJobConfigMetadata(jobConfig))

	// Print each section with a header if it is non-empty.
	sections := []printer.SectionGenerator{
		printer.ToSectionGenerator(printer.Section{Name: "Concurrency", Descs: c.prettyPrintJobConfigConcurrency(jobConfig)}),
		func() (printer.Section, error) {
			descs, err := c.prettyPrintJobConfigSchedule(ctx, cmd, jobConfig)
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
		{"Created", formatter.FormatTimeWithTimeAgo(&jobConfig.CreationTimestamp)},
	}
}

func (c *GetJobConfigCommand) prettyPrintJobConfigConcurrency(jobConfig *execution.JobConfig) [][]string {
	return [][]string{
		{"Policy", string(jobConfig.Spec.Concurrency.Policy)},
	}
}

func (c *GetJobConfigCommand) prettyPrintJobConfigSchedule(
	ctx context.Context,
	cmd *cobra.Command,
	jobConfig *execution.JobConfig,
) ([][]string, error) {
	schedule := jobConfig.Spec.Schedule
	if schedule == nil {
		return nil, nil
	}

	result := [][]string{
		{"Enabled", strconv.FormatBool(!schedule.Disabled)},
	}

	if cron := schedule.Cron; cron != nil {
		result = append(result, []string{"Cron Expression", cron.Expression})
		result = append(result, []string{"Cron Timezone", cron.Timezone})
	}

	if constraints := schedule.Constraints; constraints != nil {
		result = printer.MaybeAppendTimeAgo(result, "Not Before", constraints.NotBefore)
		result = printer.MaybeAppendTimeAgo(result, "Not After", constraints.NotAfter)
	}

	// Here we evaluate the next schedule as a convenience to the user. Note that is
	// very likely just an approximation, since many factors influence the actual
	// time that the next job is scheduled.
	if !schedule.Disabled && schedule.Cron != nil && schedule.Cron.Expression != "" {
		resp, err := c.prettyPrintNextSchedule(ctx, cmd, jobConfig)
		if err != nil {
			return nil, err
		}
		result = append(result, resp...)
	}

	return result, nil
}

func (c *GetJobConfigCommand) prettyPrintNextSchedule(
	ctx context.Context,
	cmd *cobra.Command,
	jobConfig *execution.JobConfig,
) ([][]string, error) {
	cfg := config.DefaultCronExecutionConfig.DeepCopy()

	schedule := jobConfig.Spec.Schedule
	if schedule == nil {
		return nil, nil
	}

	// Fetch the cron config in the cluster, otherwise use the default one.
	{
		newCfg := &configv1alpha1.CronExecutionConfig{}
		cfgName := configv1alpha1.CronExecutionConfigName
		if err := common.GetDynamicConfig(ctx, cmd, cfgName, cfg); err != nil {
			klog.ErrorS(err, "cannot fetch dynamic config, falling back to default", "name", cfgName)
		}
		cfg = newCfg
	}

	// Parse the next schedule.
	hashID, err := cache.MetaNamespaceKeyFunc(jobConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get namespaced name")
	}
	expr, err := cronparser.NewParser(cfg).Parse(schedule.Cron.Expression, hashID)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse cron expression: %v", schedule.Cron.Expression)
	}

	nextSchedule := "Never"
	next := expr.Next(time.Now())
	if !next.IsZero() {
		t := metav1.NewTime(next)
		nextSchedule = formatter.FormatTimeWithTimeAgo(&t)
	}

	return [][]string{{"Next Schedule", nextSchedule}}, nil
}

func (c *GetJobConfigCommand) prettyPrintJobConfigStatus(jobConfig *execution.JobConfig) [][]string {
	result := [][]string{
		{"State", string(jobConfig.Status.State)},
		{"Queued Jobs", strconv.Itoa(int(jobConfig.Status.Queued))},
		{"Active Jobs", strconv.Itoa(int(jobConfig.Status.Active))},
	}

	lastScheduled := "Never"
	if t := jobConfig.Status.LastScheduled; !t.IsZero() {
		lastScheduled = formatter.FormatTimeWithTimeAgo(t)
	}
	result = append(result, []string{"Last Scheduled", lastScheduled})

	return result
}
