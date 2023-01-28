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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
	"github.com/furiko-io/furiko/pkg/utils/sets"
)

var (
	ListJobExample = common.PrepareExample(`
# List all Jobs in current namespace.
{{.CommandName}} list job

# List all Jobs in current namespace belonging to JobConfig "daily-send-email".
{{.CommandName}} list job --for daily-send-email

# List all Jobs in JSON format.
{{.CommandName}} list job -o json`)
)

type ListJobCommand struct {
	*baseListCommand

	streams   *streams.Streams
	jobConfig string
	states    sets.Set[execution.JobState]
	output    printer.OutputFormat
	noHeaders bool
	watch     bool

	// Cached set of job UIDs that were previously filtered in.
	// If it was displayed before, we don't want to filter it out afterwards.
	// This is mostly useful for --watch.
	cachedFiltered sets.String
}

func NewListJobCommand(streams *streams.Streams) *cobra.Command {
	c := &ListJobCommand{
		baseListCommand: newBaseListCommand(),
		streams:         streams,
		states:          sets.New[execution.JobState](),
		cachedFiltered:  sets.NewString(),
	}

	cmd := &cobra.Command{
		Use:     "job",
		Aliases: []string{"jobs"},
		Short:   "Displays information about multiple Jobs.",
		Example: ListJobExample,
		PreRunE: common.PrerunWithKubeconfig,
		Args:    cobra.ExactArgs(0),
		RunE: common.RunAllE(
			c.Complete,
			c.Validate,
			c.Run,
		),
	}

	cmd.Flags().StringVar(&c.jobConfig, "for", "", "Return only jobs for the given job config.")
	cmd.Flags().Bool("active", false, "Show active jobs (i.e. Waiting/Running), otherwise show all states by default.")
	cmd.Flags().Bool("running", false, "Show running jobs, otherwise show all states by default.")
	cmd.Flags().Bool("finished", false, "Show finished jobs, otherwise show all states by default.")
	cmd.Flags().String("states", "", "Specify a comma-separated list of states to filter jobs by.")

	if err := completion.RegisterFlagCompletions(cmd, []completion.FlagCompletion{
		{FlagName: "for", Completer: &completion.ListJobConfigsCompleter{}},
		{FlagName: "states", Completer: completion.NewSliceCompleter(execution.JobStatesAll)},
	}); err != nil {
		common.Fatal(err, common.DefaultErrorExitCode)
	}

	return cmd
}

func (c *ListJobCommand) Complete(cmd *cobra.Command, args []string) error {
	if err := c.baseListCommand.Complete(cmd, args); err != nil {
		return err
	}

	c.output = common.GetOutputFormat(cmd)
	c.noHeaders = common.GetFlagBool(cmd, "no-headers")
	c.watch = common.GetFlagBool(cmd, "watch")

	// Handle --states.
	if v := common.GetFlagString(cmd, "states"); v != "" {
		flagStates := strings.Split(v, ",")
		for _, state := range flagStates {
			c.states.Insert(execution.JobState(strings.TrimSpace(state)))
		}
	}

	// Handle state boolean flags.
	if common.GetFlagBool(cmd, "active") {
		c.states.Insert(execution.JobStateWaiting, execution.JobStateRunning)
	}
	if common.GetFlagBool(cmd, "running") {
		c.states.Insert(execution.JobStateRunning)
	}
	if common.GetFlagBool(cmd, "finished") {
		c.states.Insert(execution.JobStateFinished)
	}

	// Add default states if empty.
	if c.states.Len() == 0 {
		c.states.Insert(execution.JobStatesAll...)
	}

	return nil
}

func (c *ListJobCommand) Validate(cmd *cobra.Command, args []string) error {
	// Validate states.
	for _, state := range c.states.UnsortedList() {
		if !state.IsValid() {
			return fmt.Errorf("invalid value for --states: %v is not valid, must be one of %v", state, execution.JobStatesAll)
		}
	}

	return nil
}

func (c *ListJobCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client := common.GetCtrlContext().Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := common.GetNamespace(cmd)
	if err != nil {
		return err
	}

	// Filter to specific JobConfig.
	var options metav1.ListOptions
	if c.jobConfig != "" {
		jobConfig, err := client.JobConfigs(namespace).Get(ctx, c.jobConfig, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "cannot get job config %v", c.jobConfig)
		}

		// Add to label selector.
		requirements, _ := labels.SelectorFromSet(labels.Set{
			jobconfig.LabelKeyJobConfigUID: string(jobConfig.UID),
		}).Requirements()
		c.labelSelector = c.labelSelector.Add(requirements...)
	}

	// Add selectors.
	if c.labelSelector != nil {
		options.LabelSelector = c.labelSelector.String()
	}
	if c.fieldSelector != nil {
		options.FieldSelector = c.fieldSelector.String()
	}

	jobList, err := client.Jobs(namespace).List(ctx, options)
	if err != nil {
		return errors.Wrapf(err, "cannot list jobs")
	}

	items := c.FilterJobs(jobList.Items)

	if len(items) == 0 && !c.watch {
		c.streams.Printf("No jobs found in %v namespace.\n", namespace)
		return nil
	}

	p := printer.NewTablePrinter(c.streams.Out)

	if len(items) > 0 {
		if err := c.PrintJobs(p, items); err != nil {
			return errors.Wrapf(err, "cannot print jobs")
		}
	}

	if c.watch {
		watchOptions := *options.DeepCopy()
		watchOptions.ResourceVersion = jobList.ResourceVersion
		watch, err := client.Jobs(namespace).Watch(ctx, watchOptions)
		if err != nil {
			return errors.Wrapf(err, "cannot watch jobs")
		}

		return WatchAndPrint(ctx, watch, c.FilterJob, func(job *execution.Job) error {
			return c.PrintJob(p, job)
		})
	}

	return nil
}

func (c *ListJobCommand) PrintJobs(p *printer.TablePrinter, jobs []*execution.Job) error {
	// Handle pretty print as a special case.
	if c.output == printer.OutputFormatPretty {
		c.prettyPrint(p, jobs)
		return nil
	}

	// Extract list.
	items := make([]printer.Object, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, job)
	}

	return printer.PrintObjects(execution.GVKJob, c.output, c.streams.Out, items)
}

func (c *ListJobCommand) PrintJob(p *printer.TablePrinter, job *execution.Job) error {
	// Handle pretty print as a special case.
	if c.output == printer.OutputFormatPretty {
		c.prettyPrint(p, []*execution.Job{job})
		return nil
	}

	return printer.PrintObject(execution.GVKJob, c.output, c.streams.Out, job)
}

func (c *ListJobCommand) makeJobHeader() []string {
	return []string{
		"NAME",
		"PHASE",
		"START TIME",
		"RUN TIME",
		"FINISH TIME",
	}
}

func (c *ListJobCommand) prettyPrint(p *printer.TablePrinter, jobs []*execution.Job) {
	var headers []string
	if !c.noHeaders {
		headers = c.makeJobHeader()
	}
	p.Print(headers, c.makeJobRows(jobs))
}

func (c *ListJobCommand) makeJobRows(jobs []*execution.Job) [][]string {
	rows := make([][]string, 0, len(jobs))
	for _, item := range jobs {
		rows = append(rows, c.makeJobRow(item))
	}
	return rows
}

func (c *ListJobCommand) makeJobRow(job *execution.Job) []string {
	startTime := formatter.FormatTimeAgo(job.Status.StartTime)
	runTime := ""
	finishTime := ""

	if condition := job.Status.Condition.Running; condition != nil {
		runTime = formatter.FormatTimeAgo(&condition.LatestRunningTimestamp)
	}
	if condition := job.Status.Condition.Finished; condition != nil {
		finishTime = formatter.FormatTimeAgo(&condition.FinishTimestamp)
	}

	return []string{
		job.Name,
		string(job.Status.Phase),
		startTime,
		runTime,
		finishTime,
	}
}

// FilterJobs returns a filtered list of jobs.
func (c *ListJobCommand) FilterJobs(jobs []execution.Job) []*execution.Job {
	filtered := make([]*execution.Job, 0, len(jobs))
	for _, job := range jobs {
		job := job.DeepCopy()
		if c.FilterJob(job) {
			filtered = append(filtered, job)
		}
	}
	return filtered
}

// FilterJob returns true if the job is retained in the filter.
func (c *ListJobCommand) FilterJob(job *execution.Job) bool {
	// Check if filtered in, and store in the filter cache.
	if c.filterJob(job) {
		c.cachedFiltered.Insert(string(job.UID))
	}

	// Look up exclusively in the filter cache.
	// If a job was previously filtered in, we will always allow it to pass in subsequent calls.
	return c.cachedFiltered.Has(string(job.UID))
}

// filterJob contains the actual filter logic, and returns true if the job should be filtered in.
func (c *ListJobCommand) filterJob(job *execution.Job) bool {
	return c.states.Has(job.Status.State)
}
