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

package cmd_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/cmd"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

func TestListJobCommand(t *testing.T) {
	runtimetesting.RunCommandTests(t, []runtimetesting.CommandTest{
		{
			Name: "display help",
			Args: []string{"list", "job", "--help"},
			Stdout: runtimetesting.Output{
				Contains: cmd.ListJobExample,
			},
		},
		{
			Name: "no jobs to view",
			Args: []string{"list", "job"},
			Stdout: runtimetesting.Output{
				Contains: "No jobs found in default namespace.",
			},
		},
		{
			Name:     "list a single job",
			Args:     []string{"list", "job", "-o", "name"},
			Fixtures: []runtime.Object{jobRunning},
			Stdout: runtimetesting.Output{
				Contains: "job.execution.furiko.io/job-running",
			},
		},
		{
			Name:     "can use alias",
			Args:     []string{"list", "jobs", "-o", "name"},
			Fixtures: []runtime.Object{jobRunning},
			Stdout: runtimetesting.Output{
				Contains: "job.execution.furiko.io/job-running",
			},
		},
		{
			Name:     "list job, print table",
			Args:     []string{"list", "job"},
			Fixtures: []runtime.Object{jobRunning},
			Now:      testutils.Mktime("2021-02-09T04:05:00Z"),
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					"NAME",
					"PHASE",
					jobRunning.GetName(),
					string(jobRunning.Status.Phase),
					"3m", // create time
				},
			},
		},
		{
			Name:     "list job, print table with no headers",
			Args:     []string{"list", "job", "--no-headers"},
			Fixtures: []runtime.Object{jobRunning},
			Now:      testutils.Mktime("2021-02-09T04:05:00Z"),
			Stdout: runtimetesting.Output{
				ExcludesAll: []string{
					"NAME",
					"PHASE",
				},
				ContainsAll: []string{
					jobRunning.GetName(),
					string(jobRunning.Status.Phase),
					"3m", // create time
				},
			},
		},
		{
			Name:     "list multiple jobs",
			Args:     []string{"list", "job"},
			Fixtures: []runtime.Object{jobRunning, jobFinished, jobQueued},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobRunning.GetName(),
					string(jobRunning.Status.Phase),
					jobFinished.GetName(),
					string(jobFinished.Status.Phase),
					jobQueued.GetName(),
					string(jobQueued.Status.Phase),
				},
			},
		},
		{
			Name: "cannot find job config with --for",
			Args: []string{"list", "job", "--for", adhocJobConfig.Name},
			Fixtures: []runtime.Object{
				jobRunning,
				jobFinished,
				jobQueued,
			},
			WantError: testutils.AssertErrorIsNotFound(),
		},
		{
			Name: "list only jobs belonging to jobconfig with --for",
			Args: []string{"list", "job", "--for", adhocJobConfig.Name},
			Fixtures: []runtime.Object{
				jobRunning,
				jobFinished,
				jobQueued,
				adhocJobConfig,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobRunning.GetName(),
				},
				ExcludesAll: []string{
					jobFinished.GetName(),
					jobQueued.GetName(),
				},
			},
		},
		{
			Name:     "list jobs with --active",
			Args:     []string{"list", "job", "--active"},
			Fixtures: []runtime.Object{jobRunning, jobFinished, jobQueued},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobRunning.GetName(),
				},
				ExcludesAll: []string{
					jobFinished.GetName(),
					jobQueued.GetName(),
				},
			},
		},
		{
			Name:     "list jobs with --running",
			Args:     []string{"list", "job", "--running"},
			Fixtures: []runtime.Object{jobRunning, jobFinished, jobQueued},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobRunning.GetName(),
				},
				ExcludesAll: []string{
					jobFinished.GetName(),
					jobQueued.GetName(),
				},
			},
		},
		{
			Name:     "list jobs with --finished",
			Args:     []string{"list", "job", "--finished"},
			Fixtures: []runtime.Object{jobRunning, jobFinished, jobQueued},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobFinished.GetName(),
				},
				ExcludesAll: []string{
					jobRunning.GetName(),
					jobQueued.GetName(),
				},
			},
		},
		{
			Name:     "list jobs with --active and --finished",
			Args:     []string{"list", "job", "--active", "--finished"},
			Fixtures: []runtime.Object{jobRunning, jobFinished, jobQueued},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobRunning.GetName(),
					jobFinished.GetName(),
				},
				ExcludesAll: []string{
					jobQueued.GetName(),
				},
			},
		},
		{
			Name:      "cannot list jobs with invalid --states",
			Args:      []string{"list", "job", "--states", "Invalid"},
			Fixtures:  []runtime.Object{jobRunning, jobFinished, jobQueued},
			WantError: assert.Error,
		},
		{
			Name:     "list jobs with --states",
			Args:     []string{"list", "job", "--states", "Queued"},
			Fixtures: []runtime.Object{jobRunning, jobFinished, jobQueued},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobQueued.GetName(),
				},
				ExcludesAll: []string{
					jobRunning.GetName(),
					jobFinished.GetName(),
				},
			},
		},
		{
			Name:     "list jobs with --states multiple values",
			Args:     []string{"list", "job", "--states", "Queued,Running"},
			Fixtures: []runtime.Object{jobRunning, jobFinished, jobQueued},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobQueued.GetName(),
					jobRunning.GetName(),
				},
				ExcludesAll: []string{
					jobFinished.GetName(),
				},
			},
		},
		{
			Name:     "list jobs with --states multiple values with space",
			Args:     []string{"list", "job", "--states", "Queued, Running"},
			Fixtures: []runtime.Object{jobRunning, jobFinished, jobQueued},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobQueued.GetName(),
					jobRunning.GetName(),
				},
				ExcludesAll: []string{
					jobFinished.GetName(),
				},
			},
		},
		{
			Name: "list jobs with --selector",
			Args: []string{"list", "job", "-l", "labels.furiko.io/test=value"},
			Fixtures: []runtime.Object{
				jobRunning,
				jobQueued,
				jobWithLabel,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobWithLabel.GetName(),
				},
				ExcludesAll: []string{
					jobRunning.GetName(),
					jobQueued.GetName(),
				},
			},
		},
		{
			Name: "list jobs with --selector and --running",
			Args: []string{"list", "job", "-l", "labels.furiko.io/test=value", "--running"},
			Fixtures: []runtime.Object{
				jobRunning,
				jobQueued,
				jobWithLabel,
			},
			Stdout: runtimetesting.Output{
				Contains: "No jobs found",
				ExcludesAll: []string{
					jobWithLabel.GetName(),
					jobRunning.GetName(),
					jobQueued.GetName(),
				},
			},
		},
		{
			Name: "list jobs with --selector and --states=Queued",
			Args: []string{"list", "job", "-l", "labels.furiko.io/test=value", "--states", "Queued"},
			Fixtures: []runtime.Object{
				jobRunning,
				jobQueued,
				jobWithLabel,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobWithLabel.GetName(),
				},
				ExcludesAll: []string{
					jobQueued.GetName(),
					jobRunning.GetName(),
				},
			},
		},
		{
			Name: "watch does not print headers with no jobs",
			Args: []string{"list", "job", "-w"},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Wait for a bit before cancelling the context.
				time.Sleep(time.Millisecond * 250)
				rc.Cancel()

				// Consume entire buffer.
				output, _ := rc.Console.ExpectEOF()
				assert.Len(t, output, 0)
			},
			Stdout: runtimetesting.Output{
				NumLines: pointer.Int(0),
			},
		},
		{
			Name: "watch with no jobs initially will print rows after creation",
			Args: []string{"list", "job", "-w"},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Create a new Job.
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().Jobs(jobRunning.Namespace).Create(rc.Context, jobRunning, metav1.CreateOptions{})
				assert.NoError(t, err)

				// Expect that headers and rows are printed.
				rc.Console.ExpectString("NAME")
				rc.Console.ExpectString(jobRunning.Name)

				// Cancel watch.
				rc.Cancel()
			},
			Stdout: runtimetesting.Output{
				NumLines: pointer.Int(2),
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
		{
			Name: "watch for new jobs",
			Args: []string{"list", "job", "-w"},
			Fixtures: []runtime.Object{
				jobRunning,
			},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Expect that initial output contains the specified job.
				rc.Console.ExpectString(jobRunning.Name)
				rc.Console.ExpectString(string(v1alpha1.JobRunning))

				// Create a new Job.
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().Jobs(jobFinished.Namespace).Create(rc.Context, jobFinished, metav1.CreateOptions{})
				assert.NoError(t, err)

				// Wait for the console to print updates.
				rc.Console.ExpectString(jobFinished.Name)
				rc.Console.ExpectString(string(v1alpha1.JobFailed))

				// Cancel watch.
				rc.Cancel()
			},
			Stdout: runtimetesting.Output{
				NumLines: pointer.Int(3),
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
		{
			Name: "watch jobs for updates",
			Args: []string{"list", "job", "-w"},
			Fixtures: []runtime.Object{
				jobRunning,
			},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Expect that initial output contains the specified job.
				rc.Console.ExpectString(jobRunning.Name)
				rc.Console.ExpectString(string(v1alpha1.JobRunning))

				// Update the Job's state.
				newJob := jobRunning.DeepCopy()
				newJob.Status.Phase = v1alpha1.JobSucceeded
				newJob.Status.State = v1alpha1.JobStateFinished
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().Jobs(newJob.Namespace).Update(rc.Context, newJob, metav1.UpdateOptions{})
				assert.NoError(t, err)

				// Wait for the console to print updates.
				rc.Console.ExpectString(newJob.Name)
				rc.Console.ExpectString(string(newJob.Status.Phase))

				// Cancel watch.
				rc.Cancel()
			},
			Stdout: runtimetesting.Output{
				NumLines: pointer.Int(3),
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
		{
			Name: "watch with --states",
			Args: []string{"list", "job", "-w", "--states", "Running"},
			Fixtures: []runtime.Object{
				jobQueued,
				jobRunning,
			},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Expect that initial output contains the specified job.
				rc.Console.ExpectString(jobRunning.Name)
				rc.Console.ExpectString(string(v1alpha1.JobRunning))

				// Update the Job's state.
				newJob := jobRunning.DeepCopy()
				newJob.Status.Phase = v1alpha1.JobSucceeded
				newJob.Status.State = v1alpha1.JobStateFinished
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().Jobs(newJob.Namespace).Update(rc.Context, newJob, metav1.UpdateOptions{})
				assert.NoError(t, err)

				// Wait for the console to print updates.
				rc.Console.ExpectString(newJob.Name)
				rc.Console.ExpectString(string(newJob.Status.Phase))

				// Cancel watch.
				rc.Cancel()
			},
			Stdout: runtimetesting.Output{
				NumLines: pointer.Int(3),
				ExcludesAll: []string{
					jobQueued.GetName(),
				},
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
		{
			Name: "watch with --selector and --states",
			Args: []string{"list", "job", "-w", "-l", "labels.furiko.io/test=value", "--states", "Queued"},
			Fixtures: []runtime.Object{
				jobQueued,
				jobRunning,
			},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Create a new Job.
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().Jobs(jobWithLabel.Namespace).Create(rc.Context, jobWithLabel, metav1.CreateOptions{})
				assert.NoError(t, err)

				// Expect that headers and rows are printed.
				rc.Console.ExpectString("NAME")
				rc.Console.ExpectString(jobWithLabel.Name)

				// Cancel watch.
				rc.Cancel()
			},
			Stdout: runtimetesting.Output{
				NumLines: pointer.Int(2),
				ExcludesAll: []string{
					jobQueued.GetName(),
					jobRunning.GetName(),
				},
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
		{
			Name: "watch jobs belonging to jobconfig with --for and --finished",
			Args: []string{"list", "job", "--for", adhocJobConfig.Name},
			Fixtures: []runtime.Object{
				jobRunning,
				jobFinished,
				jobQueued,
				adhocJobConfig,
			},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Update the Job's state.
				newJob := jobRunning.DeepCopy()
				newJob.Status.Phase = v1alpha1.JobSucceeded
				newJob.Status.State = v1alpha1.JobStateFinished
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().Jobs(newJob.Namespace).Update(rc.Context, newJob, metav1.UpdateOptions{})
				assert.NoError(t, err)

				// Expect that headers and rows are printed.
				rc.Console.ExpectString("NAME")
				rc.Console.ExpectString(newJob.Name)
				rc.Console.ExpectString(string(newJob.Status.Phase))

				// Cancel watch.
				rc.Cancel()
			},
			Stdout: runtimetesting.Output{
				NumLines: pointer.Int(2),
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
	})
}
