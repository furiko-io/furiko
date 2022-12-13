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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/cmd"
	"github.com/furiko-io/furiko/pkg/cli/console"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/errors"
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
			WantError: errors.AssertIsNotFound(),
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
				Matches: regexp.MustCompile(`^Job [^\s]+ created`),
			},
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
				Matches: regexp.MustCompile(`^Job [^\s]+ created`),
			},
		},
		{
			Name:     "created job with start after",
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
				Matches: regexp.MustCompile(`^Job [^\s]+ created`),
			},
		},
		{
			Name:      "created job with invalid start after",
			Args:      []string{"run", "adhoc-jobconfig", "--at", "1234"},
			Fixtures:  []runtime.Object{adhocJobConfig},
			WantError: assert.Error,
		},
		{
			Name:     "created job with start after and concurrency policy",
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
				Matches: regexp.MustCompile(`^Job [^\s]+ created`),
			},
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
				Matches: regexp.MustCompile(`^Job [^\s]+ created`),
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
				Matches: regexp.MustCompile(`Job [^\s]+ created`),
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
				Matches: regexp.MustCompile(`Job [^\s]+ created`),
			},
		},
	})
}
