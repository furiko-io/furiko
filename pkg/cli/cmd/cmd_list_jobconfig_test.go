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
	"regexp"
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
			Name: "only show job configs in the default namespace",
			Args: []string{"list", "jobconfig", "-o", "yaml"},
			Fixtures: []runtime.Object{
				periodicJobConfig,
				prodJobConfig,
			},
			Stdout: runtimetesting.Output{
				Contains: string(periodicJobConfig.UID),
				Excludes: string(prodJobConfig.UID),
			},
		},
		{
			Name: "use explicit namespace",
			Args: []string{"list", "jobconfig", "-o", "yaml", "-n", ProdNamespace},
			Fixtures: []runtime.Object{
				periodicJobConfig,
				prodJobConfig,
			},
			Stdout: runtimetesting.Output{
				Contains: string(prodJobConfig.UID),
				Excludes: string(periodicJobConfig.UID),
			},
		},
		{
			Name: "use all namespaces",
			Args: []string{"list", "jobconfig", "-o", "yaml", "-A"},
			Fixtures: []runtime.Object{
				periodicJobConfig,
				prodJobConfig,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					string(prodJobConfig.UID),
					string(periodicJobConfig.UID),
				},
			},
		},
		{
			Name: "use all namespaces, pretty print",
			Args: []string{"list", "jobconfig", "-A"},
			Fixtures: []runtime.Object{
				periodicJobConfig,
				prodJobConfig,
			},
			Stdout: runtimetesting.Output{
				MatchesAll: []*regexp.Regexp{
					regexp.MustCompile(`default\s+periodic-jobconfig`),
					regexp.MustCompile(`prod\s+periodic-jobconfig`),
				},
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
			Name: "list jobconfig with --selector",
			Args: []string{"list", "jobconfig", "-o", "name", "-l", "labels.furiko.io/test=value"},
			Fixtures: []runtime.Object{
				periodicJobConfig,
				adhocJobConfig,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					periodicJobConfig.GetName(),
				},
				ExcludesAll: []string{
					adhocJobConfig.GetName(),
				},
			},
		},
		{
			Name: "list scheduled jobconfigs with --scheduled",
			Args: []string{"list", "jobconfig", "-o", "name", "--scheduled"},
			Fixtures: []runtime.Object{
				disabledJobConfig,
				adhocJobConfig,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					disabledJobConfig.GetName(),
				},
				ExcludesAll: []string{
					adhocJobConfig.GetName(),
				},
			},
		},
		{
			Name: "don't list disabled jobconfigs with --schedule-enabled",
			Args: []string{"list", "jobconfig", "-o", "name", "--schedule-enabled"},
			Fixtures: []runtime.Object{
				disabledJobConfig,
				adhocJobConfig,
			},
			Stdout: runtimetesting.Output{
				ExcludesAll: []string{
					disabledJobConfig.GetName(),
					adhocJobConfig.GetName(),
				},
			},
		},
		{
			Name: "list adhoc-only jobconfigs with --adhoc-only",
			Args: []string{"list", "jobconfig", "-o", "name", "--adhoc-only"},
			Fixtures: []runtime.Object{
				disabledJobConfig,
				adhocJobConfig,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					adhocJobConfig.GetName(),
				},
				ExcludesAll: []string{
					disabledJobConfig.GetName(),
				},
			},
		},
		{
			Name: "cannot specify both --scheduled and --adhoc-only together",
			Args: []string{"list", "jobconfig", "-o", "name", "--scheduled", "--adhoc-only"},
			Fixtures: []runtime.Object{
				disabledJobConfig,
				adhocJobConfig,
			},
			WantError: assert.Error,
		},
		{
			Name: "cannot specify both --schedule-enabled and --adhoc-only together",
			Args: []string{"list", "jobconfig", "-o", "name", "--schedule-enabled", "--adhoc-only"},
			Fixtures: []runtime.Object{
				disabledJobConfig,
				adhocJobConfig,
			},
			WantError: assert.Error,
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
		{
			Name: "watch does not print headers with no job configs",
			Args: []string{"list", "jobconfig", "-w"},
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
			Name: "watch with no job configs initially will print rows after creation",
			Args: []string{"list", "jobconfig", "-w"},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Create a new JobConfig.
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().JobConfigs(adhocJobConfig.Namespace).Create(rc.Context, adhocJobConfig, metav1.CreateOptions{})
				assert.NoError(t, err)

				// Expect that headers and rows are printed.
				rc.Console.ExpectString("NAME")
				rc.Console.ExpectString(adhocJobConfig.Name)

				// Cancel watch.
				rc.Cancel()
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
		{
			Name: "watch for new jobconfigs",
			Args: []string{"list", "jobconfig", "-w"},
			Fixtures: []runtime.Object{
				periodicJobConfig,
			},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Expect that initial output contains the specified job config.
				rc.Console.ExpectString(periodicJobConfig.Name)
				rc.Console.ExpectString(string(v1alpha1.JobConfigReadyEnabled))

				// Create a new JobConfig.
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().JobConfigs(adhocJobConfig.Namespace).Create(rc.Context, adhocJobConfig, metav1.CreateOptions{})
				assert.NoError(t, err)

				// Wait for the console to print updates.
				rc.Console.ExpectString(adhocJobConfig.Name)
				rc.Console.ExpectString(string(v1alpha1.JobConfigReady))

				// Cancel watch.
				rc.Cancel()
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
		{
			Name: "watch jobconfigs for updates",
			Args: []string{"list", "jobconfig", "-w"},
			Fixtures: []runtime.Object{
				periodicJobConfig,
			},
			Procedure: func(t *testing.T, rc runtimetesting.RunContext) {
				// Expect that initial output contains the specified job config.
				rc.Console.ExpectString(periodicJobConfig.Name)
				rc.Console.ExpectString(string(v1alpha1.JobConfigReadyEnabled))

				// Update the JobConfig's state.
				newJobConfig := periodicJobConfig.DeepCopy()
				newJobConfig.Status.State = v1alpha1.JobConfigExecuting
				_, err := rc.CtrlContext.Clientsets().Furiko().ExecutionV1alpha1().JobConfigs(newJobConfig.Namespace).Update(rc.Context, newJobConfig, metav1.UpdateOptions{})
				assert.NoError(t, err)

				// Wait for the console to print updates.
				rc.Console.ExpectString(periodicJobConfig.Name)
				rc.Console.ExpectString(string(v1alpha1.JobConfigExecuting))

				// Cancel watch.
				rc.Cancel()
			},
			WantActions: runtimetesting.CombinedActions{
				Ignore: true,
			},
		},
		{
			Name: "print multiple cron expressions",
			Args: []string{"list", "jobconfig"},
			Fixtures: []runtime.Object{
				multiCronJobConfig,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					multiCronJobConfig.Spec.Schedule.Cron.Expressions[0],
					multiCronJobConfig.Spec.Schedule.Cron.Expressions[1],
				},
			},
		},
	})
}
