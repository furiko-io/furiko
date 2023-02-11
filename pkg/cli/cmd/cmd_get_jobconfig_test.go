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

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/cmd"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

var (
	periodicJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "periodic-jobconfig",
			Namespace: DefaultNamespace,
			UID:       testutils.MakeUID("periodic-jobconfig"),
			Labels: map[string]string{
				"labels.furiko.io/test": "value",
			},
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy: execution.ConcurrencyPolicyForbid,
			},
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "H/5 * * * *",
					Timezone:   "Asia/Singapore",
				},
			},
		},
		Status: execution.JobConfigStatus{
			State: execution.JobConfigReadyEnabled,
		},
	}

	disabledJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "periodic-jobconfig",
			Namespace: DefaultNamespace,
			UID:       testutils.MakeUID("periodic-jobconfig"),
			Labels: map[string]string{
				"labels.furiko.io/test": "value",
			},
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy: execution.ConcurrencyPolicyForbid,
			},
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "H/5 * * * *",
					Timezone:   "Asia/Singapore",
				},
				Disabled: true,
			},
		},
		Status: execution.JobConfigStatus{
			State: execution.JobConfigReadyEnabled,
		},
	}

	adhocJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "adhoc-jobconfig",
			Namespace: DefaultNamespace,
			UID:       testutils.MakeUID("adhoc-jobconfig"),
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy: execution.ConcurrencyPolicyAllow,
			},
		},
		Status: execution.JobConfigStatus{
			State: execution.JobConfigReady,
		},
	}

	jobConfigLastScheduled = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jobconfig-last-scheduled",
			Namespace: DefaultNamespace,
			UID:       testutils.MakeUID("jobconfig-last-scheduled"),
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy: execution.ConcurrencyPolicyAllow,
			},
		},
		Status: execution.JobConfigStatus{
			State:         execution.JobConfigReady,
			LastScheduled: testutils.Mkmtimep("2022-01-01T01:00:00Z"),
			LastExecuted:  testutils.Mkmtimep("2022-01-01T03:14:00Z"),
		},
	}

	multiCronJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multicron-jobconfig",
			Namespace: DefaultNamespace,
			UID:       testutils.MakeUID("multicron-jobconfig"),
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy: execution.ConcurrencyPolicyForbid,
			},
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expressions: []string{
						"H/5 10-18 * * *",
						"H 0-9,19-23 * * *",
					},
					Timezone: "Asia/Singapore",
				},
			},
		},
		Status: execution.JobConfigStatus{
			State: execution.JobConfigReadyEnabled,
		},
	}

	prodJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "periodic-jobconfig",
			Namespace: ProdNamespace,
			UID:       testutils.MakeUID("prod/periodic-jobconfig"),
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy: execution.ConcurrencyPolicyForbid,
			},
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "H/5 * * * *",
					Timezone:   "Asia/Singapore",
				},
			},
		},
		Status: execution.JobConfigStatus{
			State: execution.JobConfigReadyEnabled,
		},
	}
)

func TestGetJobConfigCommand(t *testing.T) {
	runtimetesting.RunCommandTests(t, []runtimetesting.CommandTest{
		{
			Name: "display help",
			Args: []string{"get", "jobconfig", "--help"},
			Stdout: runtimetesting.Output{
				Contains: cmd.GetJobConfigExample,
			},
		},
		{
			Name:      "need an argument",
			Args:      []string{"get", "jobconfig"},
			WantError: assert.Error,
		},
		{
			Name: "completion",
			Args: []string{cobra.ShellCompRequestCmd, "get", "jobconfig", ""},
			Fixtures: []runtime.Object{
				periodicJobConfig,
				adhocJobConfig,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					periodicJobConfig.Name,
					adhocJobConfig.Name,
				},
			},
		},
		{
			Name:     "get a single jobconfig",
			Args:     []string{"get", "jobconfig", "periodic-jobconfig", "-o", "name"},
			Fixtures: []runtime.Object{periodicJobConfig},
			Stdout: runtimetesting.Output{
				Contains: "jobconfig.execution.furiko.io/periodic-jobconfig",
			},
		},
		{
			Name:     "can use alias",
			Args:     []string{"get", "jobconfigs", "periodic-jobconfig", "-o", "name"},
			Fixtures: []runtime.Object{periodicJobConfig},
			Stdout: runtimetesting.Output{
				Contains: "jobconfig.execution.furiko.io/periodic-jobconfig",
			},
		},
		{
			Name:     "get a single jobconfig, pretty print",
			Args:     []string{"get", "jobconfig", "periodic-jobconfig"},
			Fixtures: []runtime.Object{periodicJobConfig},
			Now:      testutils.Mktime(currentTime),
			Stdout: runtimetesting.Output{
				// We expect some important information to be printed in the output.
				ContainsAll: []string{
					// Name
					periodicJobConfig.GetName(),
					// Namespace
					periodicJobConfig.GetNamespace(),
					// Current state
					string(periodicJobConfig.Status.State),
					// Cron expression
					"H/5 * * * *",
					// Cron timezone
					"Asia/Singapore",
					// Next scheduled time
					"in 2m",
				},
			},
		},
		{
			Name:     "get a single adhoc jobconfig, pretty print",
			Args:     []string{"get", "jobconfig", "adhoc-jobconfig"},
			Fixtures: []runtime.Object{adhocJobConfig},
			Now:      testutils.Mktime(currentTime),
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					adhocJobConfig.GetName(),
					adhocJobConfig.GetNamespace(),
					string(adhocJobConfig.Status.State),
				},
				MatchesAll: []*regexp.Regexp{
					// Format last scheduled correctly
					regexp.MustCompile(`Last Scheduled:\s+Never`),
				},
				ExcludesAll: []string{
					// Cannot show next schedule
					"Next Schedule",
				},
			},
		},
		{
			Name:      "get jobconfig does not exist",
			Args:      []string{"get", "jobconfig", "periodic-jobconfig"},
			WantError: testutils.AssertErrorIsNotFound(),
		},
		{
			Name: "get multiple jobconfigs",
			Args: []string{"get", "jobconfig", "periodic-jobconfig", "adhoc-jobconfig", "-o", "name"},
			Fixtures: []runtime.Object{
				periodicJobConfig,
				adhocJobConfig,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					"jobconfig.execution.furiko.io/periodic-jobconfig",
					"jobconfig.execution.furiko.io/adhoc-jobconfig",
				},
			},
		},
		{
			Name: "print multiple cron expressions",
			Args: []string{"get", "jobconfig", "multicron-jobconfig"},
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
