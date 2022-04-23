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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/cmd"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
)

var (
	periodicJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "periodic-jobconfig",
			Namespace: "default",
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

	adhocJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "adhoc-jobconfig",
			Namespace: "default",
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
			Stdout: runtimetesting.Output{
				// We expect some important information to be printed in the output.
				ContainsAll: []string{
					periodicJobConfig.GetName(),
					string(periodicJobConfig.Status.State),
					"H/5 * * * *",
					"Asia/Singapore",
				},
			},
		},
		{
			Name:      "get jobconfig does not exist",
			Args:      []string{"get", "jobconfig", "periodic-jobconfig"},
			WantError: runtimetesting.AssertErrorIsNotFound(),
		},
	})
}
