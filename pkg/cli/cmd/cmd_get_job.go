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

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/execution/util/parallel"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	stringsutils "github.com/furiko-io/furiko/pkg/utils/strings"
)

var (
	GetJobExample = common.PrepareExample(`
# Get a single Job in the current namespace.
{{.CommandName}} get job job-sample-l6dc6

# Get multiple Jobs by name in the current namespace.
{{.CommandName}} get job job-sample-l6dc6 job-sample-cdwqb

# Get a single Job in JSON format.
{{.CommandName}} get job job-sample-l6dc6 -o json`)
)

type GetJobCommand struct {
	streams *streams.Streams

	output printer.OutputFormat
}

func NewGetJobCommand(streams *streams.Streams) *cobra.Command {
	c := &GetJobCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:               "job",
		Aliases:           []string{"jobs"},
		Short:             "Displays information about one or more Jobs.",
		Example:           GetJobExample,
		PreRunE:           common.PrerunWithKubeconfig,
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completion.CompleterToCobraCompletionFunc(&completion.ListJobsCompleter{}),
		RunE: common.RunAllE(
			c.Complete,
			c.Run,
		),
	}

	return cmd
}

func (c *GetJobCommand) Complete(cmd *cobra.Command, args []string) error {
	c.output = common.GetOutputFormat(cmd)
	return nil
}

func (c *GetJobCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client := common.GetCtrlContext().Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := common.GetNamespace(cmd)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return errors.New("at least one name is required")
	}

	// Fetch all jobs.
	jobs := make([]*execution.Job, 0, len(args))
	for _, name := range args {
		job, err := client.Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "cannot get job")
		}
		jobs = append(jobs, job)
	}

	return c.PrintJobs(c.output, jobs)
}

func (c *GetJobCommand) PrintJobs(output printer.OutputFormat, jobs []*execution.Job) error {
	var detailed bool
	switch output {
	case printer.OutputFormatDetail:
		detailed = true
		fallthrough
	case printer.OutputFormatPretty:
		p := printer.NewDescriptionPrinter(c.streams.Out)
		for i, job := range jobs {
			c.prettyPrintJob(p, job, detailed)
			if i+1 < len(jobs) {
				p.Divider()
			}
		}
		return nil
	}

	// Print objects.
	items := make([]printer.Object, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, job)
	}

	return printer.PrintObjects(execution.GVKJob, output, c.streams.Out, items)
}

type section struct {
	name string
	kvs  [][]string
}

func (c *GetJobCommand) prettyPrintJob(p *printer.DescriptionPrinter, job *execution.Job, detailed bool) {
	// Print each section with a header if it is non-empty.
	sections := []section{
		{name: "Job Info", kvs: c.prettyPrintJobInfo(job)},
		{name: "Job Status", kvs: c.prettyPrintJobStatus(job)},
	}

	// Print parallel task summary.
	if job.Status.ParallelStatus != nil {
		sections = append(sections, c.prettyPrintParallelStatus(job, detailed)...)
	}

	// Show the latest task if it is non-parallel, and we are not showing all tasks.
	// The latest task alone does not make sense for parallel jobs.
	if !detailed && job.Status.ParallelStatus == nil && len(job.Status.Tasks) > 0 {
		sections = append(sections, section{
			name: "Latest Task",
			kvs:  c.prettyPrintJobTaskStatus(job.Status.Tasks[len(job.Status.Tasks)-1]),
		})
	}

	// Print all task statuses, including if it is parallel.
	if detailed {
		for i, task := range job.Status.Tasks {
			sections = append(sections, section{
				name: fmt.Sprintf("Task %v Detail", i+1),
				kvs:  c.prettyPrintJobTaskStatus(task),
			})
		}
	}

	var printed int
	for _, section := range sections {
		if len(section.kvs) == 0 {
			continue
		}
		if printed > 0 {
			p.NewLine()
		}
		p.Header(section.name)
		p.Descriptions(section.kvs)
		printed++
	}
}

func (c *GetJobCommand) prettyPrintJobInfo(job *execution.Job) [][]string {
	result := [][]string{
		{"Name", job.Name},
		{"Namespace", job.Namespace},
		{"Type", string(job.Spec.Type)},
		{"Created", formatter.FormatTimeWithTimeAgo(&job.CreationTimestamp)},
	}

	if ref := metav1.GetControllerOf(job); ref != nil && ref.Kind == execution.KindJobConfig {
		result = append(result, []string{"Job Config", ref.Name})
	}

	if sp := job.Spec.StartPolicy; sp != nil {
		if !sp.StartAfter.IsZero() {
			result = append(result, []string{"Start After", formatter.FormatTimeWithTimeAgo(sp.StartAfter)})
		}
		if sp.ConcurrencyPolicy != "" {
			result = append(result, []string{"Concurrency Policy", string(sp.ConcurrencyPolicy)})
		}
	}

	return result
}

func (c *GetJobCommand) prettyPrintJobStatus(job *execution.Job) [][]string {
	result := [][]string{
		{"State", string(job.Status.State)},
		{"Phase", string(job.Status.Phase)},
	}
	result = printer.MaybeAppendTimeAgo(result, "Started", job.Status.StartTime)
	if !job.Status.StartTime.IsZero() {
		result = append(result, []string{"Tasks", c.formatJobTaskStatus(job.Status.Tasks)})
	}
	if status := job.Status.Condition.Running; status != nil {
		if !status.LatestRunningTimestamp.IsZero() {
			result = append(result, []string{
				"Run Duration",
				stringsutils.Capitalize(formatter.FormatDuration(ktime.Now().Sub(status.LatestRunningTimestamp.Time))),
			})
		}
		if status.TerminatingTasks > 0 {
			result = append(result, []string{"Terminating Tasks", formatter.FormatInt64(status.TerminatingTasks)})
		}
	}
	if status := job.Status.Condition.Finished; status != nil {
		if !status.LatestRunningTimestamp.IsZero() {
			result = append(result, []string{
				"Run Duration",
				stringsutils.Capitalize(formatter.FormatDuration(status.FinishTimestamp.Sub(status.LatestRunningTimestamp.Time))),
			})
		}
		result = append(result, []string{"Result", string(status.Result)})
	}
	if reason, message, ok := c.getReasonMessage(job); ok {
		if reason != "" {
			result = append(result, []string{"Reason", reason})
		}
		if message != "" {
			result = append(result, []string{"Message", message})
		}
	}
	return result
}

func (c *GetJobCommand) prettyPrintParallelStatus(job *execution.Job, detailed bool) []section {
	status := job.Status.ParallelStatus
	if status == nil {
		return nil
	}

	sections := []section{
		{
			name: "Parallel Task Summary",
			kvs:  c.prettyPrintParallelTaskSummary(job.Spec.Template.Parallelism, *status),
		},
	}

	// Print all indexes.
	if detailed {
		for i, index := range status.Indexes {
			section := section{
				name: fmt.Sprintf("Parallel Task Group %v Detail", i+1),
				kvs: [][]string{
					{"Group Hash", index.Hash},
				},
			}
			section.kvs = append(section.kvs, c.prettyPrintParallelIndex(index.Index)...)
			section.kvs = append(section.kvs,
				[]string{"Created Tasks", formatter.FormatInt64(index.CreatedTasks)},
				[]string{"State", string(index.State)},
			)
			if index.Result != "" {
				section.kvs = append(section.kvs, []string{"Result", string(index.Result)})
			}
			sections = append(sections, section)
		}
	}

	return sections
}

func (c *GetJobCommand) prettyPrintParallelTaskSummary(
	spec *execution.ParallelismSpec,
	status execution.ParallelStatus,
) [][]string {
	result := [][]string{
		{"Complete", formatter.FormatBool(status.Complete)},
		{"Completion Strategy", string(spec.GetCompletionStrategy())},
	}
	if successful := status.Successful; successful != nil {
		result = append(result, []string{"Successful", formatter.FormatBool(*successful)})
	}
	result = append(result, [][]string{
		{"Status", c.formatParallelTaskSummaryStatus(status)},
	}...)
	return result
}

func (c *GetJobCommand) prettyPrintParallelIndex(index execution.ParallelIndex) [][]string {
	switch {
	case index.IndexNumber != nil:
		return [][]string{{"Index Number", formatter.FormatInt64(*index.IndexNumber)}}
	case index.IndexKey != "":
		return [][]string{{"Index Key", index.IndexKey}}
	case len(index.MatrixValues) > 0:
		vals := make([][]string, 0, len(index.MatrixValues))
		for k, v := range index.MatrixValues {
			vals = append(vals, []string{fmt.Sprintf("Matrix Key %v", k), v})
		}
		return vals
	}
	return nil
}

func (c *GetJobCommand) prettyPrintJobTaskStatus(task execution.TaskRef) [][]string {
	result := [][]string{
		{"Name", task.Name},
		{"Created", formatter.FormatTimeWithTimeAgo(&task.CreationTimestamp)},
		{"State", string(task.Status.State)},
	}
	result = printer.MaybeAppendTimeAgo(result, "Started", task.RunningTimestamp)
	result = printer.MaybeAppendTimeAgo(result, "Finished", task.FinishTimestamp)
	if task.Status.Reason != "" {
		result = append(result, []string{"Reason", task.Status.Reason})
	}
	if task.Status.Message != "" {
		result = append(result, []string{"Message", task.Status.Message})
	}

	return result
}

func (c *GetJobCommand) getReasonMessage(job *execution.Job) (string, string, bool) {
	if status := job.Status.Condition.Queueing; status != nil {
		return status.Reason, status.Message, true
	}
	if status := job.Status.Condition.Waiting; status != nil {
		return status.Reason, status.Message, true
	}
	if status := job.Status.Condition.Finished; status != nil {
		return status.Reason, status.Message, true
	}
	return "", "", false
}

func (c *GetJobCommand) formatJobTaskStatus(tasks []execution.TaskRef) string {
	var created, starting, running, killing, terminated, succeeded, failed, killed int64
	for _, task := range tasks {
		switch task.Status.State {
		case execution.TaskStarting:
			starting++
		case execution.TaskRunning:
			running++
		case execution.TaskKilling:
			killing++
		case execution.TaskTerminated:
			switch task.Status.Result {
			case execution.TaskSucceeded:
				succeeded++
			case execution.TaskFailed:
				failed++
			case execution.TaskKilled:
				killed++
			default:
				terminated++
			}
		default:
			created++
		}
	}

	return c.formatCounters([]counter{
		{label: "Created", count: created},
		{label: "Starting", count: starting},
		{label: "Running", count: running, showIfZero: true},
		{label: "Killing", count: killing},
		{label: "Succeeded", count: succeeded},
		{label: "Failed", count: failed},
		{label: "Killed", count: killed},
		{label: "Terminated", count: terminated},
	}, false)
}

func (c *GetJobCommand) formatParallelTaskSummaryStatus(status execution.ParallelStatus) string {
	var notCreated, created, starting, running, retryBackoff, terminated int64
	for _, index := range status.Indexes {
		switch index.State {
		case execution.IndexNotCreated:
			notCreated++
		case execution.IndexStarting:
			starting++
		case execution.IndexRunning:
			running++
		case execution.IndexRetryBackoff:
			retryBackoff++
		case execution.IndexTerminated:
			terminated++
		default:
			created++
		}
	}

	counters := parallel.GetParallelStatusCounters(status.Indexes)

	return c.formatCounters([]counter{
		{label: "Pending Creation", count: notCreated},
		{label: "Created", count: created},
		{label: "Starting", count: starting},
		{label: "Running", count: running, showIfZero: true},
		{label: "Retry Backoff", count: retryBackoff},
		// Show Succeeded/Failed instead of Terminated.
		{label: "Succeeded", count: counters.Succeeded},
		{label: "Failed", count: counters.Failed},
	}, false)
}

type counter struct {
	label      string
	count      int64
	showIfZero bool
}

func (c *GetJobCommand) formatCounters(counters []counter, showZero bool) string {
	strs := make([]string, 0, len(counters))
	for _, c := range counters {
		if c.count == 0 && !showZero && !c.showIfZero {
			continue
		}
		strs = append(strs, fmt.Sprintf("%v %v", c.count, c.label))
	}
	return strings.Join(strs, " / ")
}
