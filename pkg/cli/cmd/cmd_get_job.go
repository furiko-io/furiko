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
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	stringsutils "github.com/furiko-io/furiko/pkg/utils/strings"
)

type GetJobCommand struct{}

func NewGetJobCommand(ctx context.Context) *cobra.Command {
	c := &GetJobCommand{}
	cmd := &cobra.Command{
		Use:     "job",
		Aliases: []string{"jobs"},
		Short:   "Displays information about one or more Jobs.",
		Example: `  # Get a single Job in the current namespace.
  furiko get job job-sample-l6dc6

  # Get multiple Jobs by name in the current namespace.
  furiko get job job-sample-l6dc6 job-sample-cdwqb

  # Get a single Job in JSON format.
  furiko get job job-sample-l6dc6 -o json`,
		PreRunE: PrerunWithKubeconfig,
		Args:    cobra.MinimumNArgs(1),
		RunE:    ToRunE(ctx, c),
	}

	return cmd
}

func (c *GetJobCommand) Run(ctx context.Context, cmd *cobra.Command, args []string) error {
	client := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := GetNamespace(cmd)
	if err != nil {
		return err
	}
	output, err := GetOutputFormat(cmd)
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
			return errors.Wrapf(err, "cannot list jobs")
		}
		job.SetGroupVersionKind(execution.GVKJob)
		jobs = append(jobs, job)
	}

	return c.PrintJobs(output, jobs)
}

func (c *GetJobCommand) PrintJobs(output OutputFormat, jobs []*execution.Job) error {
	// Handle pretty print as a special case.
	if output == OutputFormatPretty {
		p := NewStdoutDescriptionPrinter()
		for i, job := range jobs {
			c.prettyPrintJob(p, job)
			if i+1 < len(jobs) {
				p.Divider()
			}
		}
		return nil
	}

	// Print items.
	items := make([]runtime.Object, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, job)
	}
	return PrintItems(output, os.Stdout, items)
}

func (c *GetJobCommand) prettyPrintJob(p *DescriptionPrinter, job *execution.Job) {
	// Print the metadata.
	p.Header("Job Metadata")
	p.Descriptions(c.prettyPrintJobMetadata(job))

	// Print each section with a header if it is non-empty.
	sections := []struct {
		name string
		kvs  [][]string
	}{
		{name: "Job Spec", kvs: c.prettyPrintJobSpec(job)},
		{name: "Job Status", kvs: c.prettyPrintJobStatus(job)},
		{name: "Latest Task", kvs: c.prettyPrintJobLatestTask(job)},
	}

	for _, section := range sections {
		if len(section.kvs) > 0 {
			p.NewLine()
			p.Header(section.name)
			p.Descriptions(section.kvs)
		}
	}
}

func (c *GetJobCommand) prettyPrintJobMetadata(job *execution.Job) [][]string {
	result := [][]string{
		{"Name", job.Name},
		{"Namespace", job.Namespace},
		{"Created", FormatTimeWithTimeAgo(&job.CreationTimestamp)},
	}

	if ref := metav1.GetControllerOf(job); ref != nil && ref.Kind == execution.KindJobConfig {
		result = append(result, []string{"Job Config", ref.Name})
	}

	return result
}

func (c *GetJobCommand) prettyPrintJobSpec(job *execution.Job) [][]string {
	var result [][]string
	if sp := job.Spec.StartPolicy; sp != nil {
		if !sp.StartAfter.IsZero() {
			result = append(result, []string{"Start After", FormatTimeWithTimeAgo(sp.StartAfter)})
		}
		if sp.ConcurrencyPolicy != "" {
			result = append(result, []string{"Concurrency Policy", string(sp.ConcurrencyPolicy)})
		}
	}
	return result
}

func (c *GetJobCommand) prettyPrintJobStatus(job *execution.Job) [][]string {
	result := [][]string{
		{"Phase", string(job.Status.Phase)},
	}
	result = MaybeAppendTimeAgo(result, "Started", job.Status.StartTime)
	if job.Status.CreatedTasks > 0 {
		result = append(result, []string{"Created Tasks", strconv.Itoa(int(job.Status.CreatedTasks))})
	}
	if status := job.Status.Condition.Finished; status != nil {
		if !status.StartedAt.IsZero() {
			result = append(result, []string{
				"Run Duration",
				stringsutils.Capitalize(FormatDuration(status.FinishedAt.Sub(status.StartedAt.Time))),
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

func (c *GetJobCommand) prettyPrintJobLatestTask(job *execution.Job) [][]string {
	if len(job.Status.Tasks) == 0 {
		return nil
	}

	task := job.Status.Tasks[len(job.Status.Tasks)-1]
	result := [][]string{
		{"Name", task.Name},
		{"Created", FormatTimeWithTimeAgo(&task.CreationTimestamp)},
		{"State", string(task.Status.State)},
	}
	result = MaybeAppendTimeAgo(result, "Started", task.RunningTimestamp)
	result = MaybeAppendTimeAgo(result, "Finished", task.FinishTimestamp)
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
