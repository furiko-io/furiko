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

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/furiko-io/furiko/pkg/cli/cmd"
	"github.com/furiko-io/furiko/pkg/cli/formatter"
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
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobRunning.GetName(),
					string(jobRunning.Status.Phase),
					formatter.FormatTimeAgo(testutils.Mkmtimep(taskCreateTime)),
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
	})
}
