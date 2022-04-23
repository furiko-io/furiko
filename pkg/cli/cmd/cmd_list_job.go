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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
)

var (
	ListJobExample = PrepareExample(`
# List all Jobs in current namespace.
{{.CommandName}} list job

# List all Jobs in current namespace belonging to JobConfig "daily-send-email".
{{.CommandName}} list job --for daily-send-email

# List all Jobs in JSON format.
{{.CommandName}} list job -o json`)
)

type ListJobCommand struct {
	streams   genericclioptions.IOStreams
	jobConfig string
}

func NewListJobCommand(streams genericclioptions.IOStreams) *cobra.Command {
	c := &ListJobCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:     "job",
		Aliases: []string{"jobs"},
		Short:   "Displays information about multiple Jobs.",
		Example: ListJobExample,
		PreRunE: PrerunWithKubeconfig,
		Args:    cobra.ExactArgs(0),
		RunE:    c.Run,
	}

	cmd.Flags().StringVar(&c.jobConfig, "for", "", "Return only jobs for the given job config.")

	return cmd
}

func (c *ListJobCommand) Run(cmd *cobra.Command, args []string) error {
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

	// Filter to specific JobConfig.
	var options metav1.ListOptions
	if c.jobConfig != "" {
		jobConfig, err := client.JobConfigs(namespace).Get(ctx, c.jobConfig, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "cannot get job config %v", c.jobConfig)
		}

		// Fetch by label.
		options.LabelSelector = labels.SelectorFromSet(labels.Set{
			jobconfig.LabelKeyJobConfigUID: string(jobConfig.UID),
		}).String()
	}

	jobList, err := client.Jobs(namespace).List(ctx, options)
	if err != nil {
		return errors.Wrapf(err, "cannot list jobs")
	}

	if len(jobList.Items) == 0 {
		_, _ = fmt.Fprintf(c.streams.Out, "No jobs found in %v namespace.\n", namespace)
		return nil
	}

	return c.PrintJobs(output, jobList)
}

func (c *ListJobCommand) PrintJobs(output printer.OutputFormat, jobList *execution.JobList) error {
	// Handle pretty print as a special case.
	if output == printer.OutputFormatPretty {
		c.prettyPrint(jobList.Items)
		return nil
	}

	// Extract list.
	items := make([]printer.Object, 0, len(jobList.Items))
	for _, job := range jobList.Items {
		job := job
		items = append(items, &job)
	}

	return printer.PrintObjects(execution.GVKJob, output, c.streams.Out, items)
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

func (c *ListJobCommand) prettyPrint(jobs []execution.Job) {
	p := printer.NewTablePrinter(c.streams.Out)
	p.Print(c.makeJobHeader(), c.makeJobRows(jobs))
}

func (c *ListJobCommand) makeJobRows(jobs []execution.Job) [][]string {
	rows := make([][]string, 0, len(jobs))
	for _, item := range jobs {
		item := item
		rows = append(rows, c.makeJobRow(&item))
	}
	return rows
}

func (c *ListJobCommand) makeJobRow(job *execution.Job) []string {
	startTime := formatter.FormatTimeAgo(job.Status.StartTime)
	runTime := ""
	finishTime := ""

	if condition := job.Status.Condition.Running; condition != nil {
		runTime = formatter.FormatTimeAgo(&condition.StartedAt)
	}
	if condition := job.Status.Condition.Finished; condition != nil {
		finishTime = formatter.FormatTimeAgo(&condition.FinishedAt)
	}

	return []string{
		job.Name,
		string(job.Status.Phase),
		startTime,
		runTime,
		finishTime,
	}
}
