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
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().Jobs(newJob.Namespace).Update(rc.Context, newJob, metav1.UpdateOptions{})
				assert.NoError(t, err)

				// Wait for the console to print updates.
				rc.Console.ExpectString(newJob.Name)
				rc.Console.ExpectString(string(newJob.Status.Phase))

				// Cancel watch.
				rc.Cancel()
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
	})
}
