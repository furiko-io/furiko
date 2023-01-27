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
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

func TestListJobConfigCommand(t *testing.T) {
	runtimetesting.RunCommandTests(t, []runtimetesting.CommandTest{
		{
			Name: "display help",
			Args: []string{"list", "jobconfig", "--help"},
			Stdout: runtimetesting.Output{
				Contains: cmd.ListJobConfigExample,
			},
		},
		{
			Name: "no jobs to view",
			Args: []string{"list", "jobconfig"},
			Stdout: runtimetesting.Output{
				Contains: "No job configs found in default namespace.",
			},
		},
		{
			Name:     "list a single jobconfig",
			Args:     []string{"list", "jobconfig", "-o", "name"},
			Fixtures: []runtime.Object{periodicJobConfig},
			Stdout: runtimetesting.Output{
				Contains: "jobconfig.execution.furiko.io/periodic-jobconfig",
			},
		},
		{
			Name:     "can use alias",
			Args:     []string{"list", "jobconfigs", "-o", "name"},
			Fixtures: []runtime.Object{periodicJobConfig},
			Stdout: runtimetesting.Output{
				Contains: "jobconfig.execution.furiko.io/periodic-jobconfig",
			},
		},
		{
			Name:     "list jobconfig, print table",
			Args:     []string{"list", "jobconfig"},
			Fixtures: []runtime.Object{periodicJobConfig},
			Stdout: runtimetesting.Output{
				// We expect some important information to be printed in the output.
				ContainsAll: []string{
					"NAME",
					"STATE",
					string(periodicJobConfig.Status.State),
					"H/5 * * * *",
				},
			},
		},
		{
			Name:     "list jobconfig, print table with no headers",
			Args:     []string{"list", "jobconfig", "--no-headers"},
			Fixtures: []runtime.Object{periodicJobConfig},
			Stdout: runtimetesting.Output{
				ExcludesAll: []string{
					"NAME",
					"STATE",
				},
				ContainsAll: []string{
					string(periodicJobConfig.Status.State),
					"H/5 * * * *",
				},
			},
		},
		{
			Name:     "list multiple jobconfigs",
			Args:     []string{"list", "jobconfig"},
			Fixtures: []runtime.Object{periodicJobConfig, adhocJobConfig},
			Stdout: runtimetesting.Output{
				// We expect some important information to be printed in the output.
				ContainsAll: []string{
					string(periodicJobConfig.Status.State),
					string(adhocJobConfig.Status.State),
					"H/5 * * * *",
				},
			},
		},
		{
			Name:     "show LastExecuted and LastScheduled",
			Args:     []string{"list", "jobconfig"},
			Fixtures: []runtime.Object{jobConfigLastScheduled},
			Now:      testutils.Mktime("2022-01-01T10:15:00Z"),
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					"9h", // last scheduled
					"7h", // last executed
				},
			},
		},
	})
}
