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

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/cmd"
	"github.com/furiko-io/furiko/pkg/cli/console"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

var (
	parameterizableJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "parameterizable-jobconfig",
			Namespace: DefaultNamespace,
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy: execution.ConcurrencyPolicyAllow,
			},
			Option: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type:     execution.OptionTypeString,
						Name:     "name",
						Label:    "Full Name",
						Required: true,
						String: &execution.StringOptionConfig{
							Default:    "Example User",
							TrimSpaces: true,
						},
					},
				},
			},
		},
		Status: execution.JobConfigStatus{
			State: execution.JobConfigReady,
		},
	}

	parameterizableJobConfigWithRequired = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "parameterizable-jobconfig",
			Namespace: DefaultNamespace,
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy: execution.ConcurrencyPolicyAllow,
			},
			Option: &execution.OptionSpec{
				Options: []execution.Option{
					{
						Type:     execution.OptionTypeString,
						Name:     "name",
						Label:    "Full Name",
						Required: true,
					},
				},
			},
		},
		Status: execution.JobConfigStatus{
			State: execution.JobConfigReady,
		},
	}
)

var (
	adhocJobCreated = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "adhoc-jobconfig-",
			Namespace:    DefaultNamespace,
		},
		Spec: execution.JobSpec{
			ConfigName:  "adhoc-jobconfig",
			StartPolicy: &execution.StartPolicySpec{},
		},
	}

	adhocJobCreatedWithCustomName = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "adhoc-run-xyz",
			Namespace: DefaultNamespace,
		},
		Spec: execution.JobSpec{
			ConfigName:  "adhoc-jobconfig",
			StartPolicy: &execution.StartPolicySpec{},
		},
	}

	adhocJobCreatedWithCustomGenerateName = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "adhoc-run-",
			Namespace:    DefaultNamespace,
		},
		Spec: execution.JobSpec{
			ConfigName:  "adhoc-jobconfig",
			StartPolicy: &execution.StartPolicySpec{},
		},
	}

	adhocJobCreatedWithConcurrencyPolicy = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "adhoc-jobconfig-",
			Namespace:    DefaultNamespace,
		},
		Spec: execution.JobSpec{
			ConfigName: "adhoc-jobconfig",
			StartPolicy: &execution.StartPolicySpec{
				ConcurrencyPolicy: execution.ConcurrencyPolicyEnqueue,
			},
		},
	}

	adhocJobCreatedWithStartAfter = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "adhoc-jobconfig-",
			Namespace:    DefaultNamespace,
		},
		Spec: execution.JobSpec{
			ConfigName: "adhoc-jobconfig",
			StartPolicy: &execution.StartPolicySpec{
				ConcurrencyPolicy: execution.ConcurrencyPolicyEnqueue,
				StartAfter:        testutils.Mkmtimep(startTime),
			},
		},
	}

	adhocJobCreatedWithStartAfterAllow = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "adhoc-jobconfig-",
			Namespace:    DefaultNamespace,
		},
		Spec: execution.JobSpec{
			ConfigName: "adhoc-jobconfig",
			StartPolicy: &execution.StartPolicySpec{
				ConcurrencyPolicy: execution.ConcurrencyPolicyAllow,
				StartAfter:        testutils.Mkmtimep(startTime),
			},
		},
	}

	parameterizableJobCreatedWithDefaultOptionValues = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "parameterizable-jobconfig-",
			Namespace:    DefaultNamespace,
		},
		Spec: execution.JobSpec{
			ConfigName:  "parameterizable-jobconfig",
			StartPolicy: &execution.StartPolicySpec{},
		},
	}

	parameterizableJobCreatedWithCustomOptionValues = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "parameterizable-jobconfig-",
			Namespace:    DefaultNamespace,
		},
		Spec: execution.JobSpec{
			ConfigName:   "parameterizable-jobconfig",
			StartPolicy:  &execution.StartPolicySpec{},
			OptionValues: `{"name":"John Smith"}`,
		},
	}
)

func TestRunCommand(t *testing.T) {
	runtimetesting.RunCommandTests(t, []runtimetesting.CommandTest{
		{
			Name: "display help",
			Args: []string{"run", "--help"},
			Stdout: runtimetesting.Output{
				Contains: cmd.RunExample,
			},
		},
		{
			Name:      "need an argument",
			Args:      []string{"run"},
			WantError: assert.Error,
		},
		{
			Name:      "jobconfig does not exist",
			Args:      []string{"run", "adhoc-jobconfig"},
			WantError: testutils.AssertErrorIsNotFound(),
		},
		{
			Name:     "created job",
			Args:     []string{"run", "adhoc-jobconfig"},
			Fixtures: []runtime.Object{adhocJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, adhocJobCreated),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches: regexp.MustCompile(`^Job \S+ created`),
			},
		},
		{
			Name:     "created job with custom name",
			Args:     []string{"run", "adhoc-jobconfig", "--name", adhocJobCreatedWithCustomName.Name},
			Fixtures: []runtime.Object{adhocJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, adhocJobCreatedWithCustomName),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches: regexp.MustCompile(`^Job \S+ created`),
			},
		},
		{
			Name:     "created job with custom generateName",
			Args:     []string{"run", "adhoc-jobconfig", "--generate-name", adhocJobCreatedWithCustomGenerateName.GenerateName},
			Fixtures: []runtime.Object{adhocJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, adhocJobCreatedWithCustomGenerateName),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches: regexp.MustCompile(`^Job \S+ created`),
			},
		},
		{
			Name:      "cannot specify --name and --generate-name together",
			Args:      []string{"run", "adhoc-jobconfig", "--name", adhocJobCreatedWithCustomName.Name, "--generate-name", adhocJobCreatedWithCustomGenerateName.GenerateName},
			Fixtures:  []runtime.Object{adhocJobConfig},
			WantError: assert.Error,
		},
		{
			Name:     "created job with concurrency policy",
			Args:     []string{"run", "adhoc-jobconfig", "--concurrency-policy", "Enqueue"},
			Fixtures: []runtime.Object{adhocJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, adhocJobCreatedWithConcurrencyPolicy),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches: regexp.MustCompile(`^Job \S+ created`),
			},
		},
		{
			Name:     "created job with enqueue",
			Args:     []string{"run", "adhoc-jobconfig", "--enqueue"},
			Fixtures: []runtime.Object{adhocJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, adhocJobCreatedWithConcurrencyPolicy),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches: regexp.MustCompile(`^Job \S+ created`),
			},
		},
		{
			Name:      "cannot specify both --concurrency-policy and --enqueue",
			Args:      []string{"run", "adhoc-jobconfig", "--enqueue", "--concurrency-policy", "Allow"},
			Fixtures:  []runtime.Object{adhocJobConfig},
			WantError: assert.Error,
		},
		{
			Name:     "created job with --at",
			Args:     []string{"run", "adhoc-jobconfig", "--at", startTime},
			Fixtures: []runtime.Object{adhocJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, adhocJobCreatedWithStartAfter),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches: regexp.MustCompile(`^Job \S+ created`),
			},
		},
		{
			Name:      "cannot create job with invalid --at",
			Args:      []string{"run", "adhoc-jobconfig", "--at", "1234"},
			Fixtures:  []runtime.Object{adhocJobConfig},
			WantError: assert.Error,
		},
		{
			Name:     "created job with --at and --concurrency-policy",
			Args:     []string{"run", "adhoc-jobconfig", "--at", startTime, "--concurrency-policy", "Allow"},
			Fixtures: []runtime.Object{adhocJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, adhocJobCreatedWithStartAfterAllow),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches: regexp.MustCompile(`^Job \S+ created`),
			},
		},
		{
			Name:     "created job with --after",
			Args:     []string{"run", "adhoc-jobconfig", "--after", "3h"},
			Now:      testutils.Mktime(startTime).Add(-3 * time.Hour),
			Fixtures: []runtime.Object{adhocJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, adhocJobCreatedWithStartAfter),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches: regexp.MustCompile(`^Job \S+ created`),
			},
		},
		{
			Name:      "cannot create job with invalid --after",
			Args:      []string{"run", "adhoc-jobconfig", "--after", "abcd"},
			Now:       testutils.Mktime(startTime).Add(-3 * time.Hour),
			Fixtures:  []runtime.Object{adhocJobConfig},
			WantError: assert.Error,
		},
		{
			Name:      "cannot specify both --at and --after",
			Args:      []string{"run", "adhoc-jobconfig", "--at", startTime, "--after", "3h"},
			Now:       testutils.Mktime(startTime).Add(-3 * time.Hour),
			Fixtures:  []runtime.Object{adhocJobConfig},
			WantError: assert.Error,
		},
		{
			Name:     "created job with default option values",
			Args:     []string{"run", "parameterizable-jobconfig", "--use-default-options"},
			Fixtures: []runtime.Object{parameterizableJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, parameterizableJobCreatedWithDefaultOptionValues),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches: regexp.MustCompile(`^Job \S+ created`),
			},
		},
		{
			Name:     "created job with prompt input",
			Args:     []string{"run", "parameterizable-jobconfig"},
			Fixtures: []runtime.Object{parameterizableJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, parameterizableJobCreatedWithCustomOptionValues),
					},
				},
			},
			Stdin: runtimetesting.Input{
				Procedure: func(c *console.Console) {
					c.ExpectString("Full Name")
					c.SendLine("John Smith")
				},
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					"Please input option values.",
					"Full Name",
				},
				Matches: regexp.MustCompile(`Job \S+ created`),
			},
		},
		{
			Name:     "prompt stdin input for required option",
			Args:     []string{"run", "parameterizable-jobconfig", "--use-default-options"},
			Fixtures: []runtime.Object{parameterizableJobConfigWithRequired},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, parameterizableJobCreatedWithCustomOptionValues),
					},
				},
			},
			Stdin: runtimetesting.Input{
				Procedure: func(c *console.Console) {
					c.ExpectString("Full Name")
					c.SendLine("John Smith")
				},
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					"Please input option values.",
					"Full Name",
				},
				Matches: regexp.MustCompile(`Job \S+ created`),
			},
		},
		{
			Name:      "cannot parse --option-values as JSON",
			Args:      []string{"run", "parameterizable-jobconfig", "--use-default-options", "--option-values", `{"invalid`},
			Fixtures:  []runtime.Object{parameterizableJobConfigWithRequired},
			WantError: assert.Error,
		},
		{
			Name:     "don't prompt stdin input if required option is passed via --option-values",
			Args:     []string{"run", "parameterizable-jobconfig", "--use-default-options", "--option-values", `{"name": "John Smith"}`},
			Fixtures: []runtime.Object{parameterizableJobConfigWithRequired},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, parameterizableJobCreatedWithCustomOptionValues),
					},
				},
			},
			Stdout: runtimetesting.Output{
				ExcludesAll: []string{
					"Please input option values.",
					"Full Name",
				},
				Matches: regexp.MustCompile(`Job \S+ created`),
			},
		},
		{
			Name:     "override default values from --use-default-options with --option-values",
			Args:     []string{"run", "parameterizable-jobconfig", "--use-default-options", "--option-values", `{"name": "John Smith"}`},
			Fixtures: []runtime.Object{parameterizableJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, parameterizableJobCreatedWithCustomOptionValues),
					},
				},
			},
			Stdout: runtimetesting.Output{
				ExcludesAll: []string{
					"Please input option values.",
					"Full Name",
				},
				Matches: regexp.MustCompile(`Job \S+ created`),
			},
		},
		{
			Name:     "did not match any options from key in --option-values",
			Args:     []string{"run", "parameterizable-jobconfig", "--use-default-options", "--option-values", `{"mismatched": "value"}`},
			Fixtures: []runtime.Object{parameterizableJobConfig},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewCreateJobAction(DefaultNamespace, parameterizableJobCreatedWithDefaultOptionValues),
					},
				},
			},
			Stdout: runtimetesting.Output{
				ExcludesAll: []string{
					"Please input option values.",
					"Full Name",
				},
				Matches: regexp.MustCompile(`Job \S+ created`),
			},
		},
	})
}
