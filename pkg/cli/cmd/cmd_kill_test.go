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
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	killTime  = "2021-02-09T04:04:04Z"
	killTime2 = "2021-02-09T04:04:44Z"
)

var (
	runningJob = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-job",
			Namespace: DefaultNamespace,
		},
		Spec: execution.JobSpec{},
		Status: execution.JobStatus{
			Phase: execution.JobRunning,
			Condition: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					LatestCreationTimestamp: testutils.Mkmtime(taskCreateTime),
					LatestRunningTimestamp:  testutils.Mkmtime(taskLaunchTime),
				},
			},
			StartTime:    testutils.Mkmtimep(startTime),
			CreatedTasks: 1,
		},
	}

	runningJobKilled = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-job",
			Namespace: DefaultNamespace,
		},
		Spec: execution.JobSpec{
			KillTimestamp: testutils.Mkmtimep(killTime),
		},
		Status: execution.JobStatus{
			Phase: execution.JobRunning,
			Condition: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					LatestCreationTimestamp: testutils.Mkmtime(taskCreateTime),
					LatestRunningTimestamp:  testutils.Mkmtime(taskLaunchTime),
				},
			},
			StartTime:    testutils.Mkmtimep(startTime),
			CreatedTasks: 1,
		},
	}

	runningJobKilledNewTimestamp = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-job",
			Namespace: DefaultNamespace,
		},
		Spec: execution.JobSpec{
			KillTimestamp: testutils.Mkmtimep(killTime2),
		},
		Status: execution.JobStatus{
			Phase: execution.JobRunning,
			Condition: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					LatestCreationTimestamp: testutils.Mkmtime(taskCreateTime),
					LatestRunningTimestamp:  testutils.Mkmtime(taskLaunchTime),
				},
			},
			StartTime:    testutils.Mkmtimep(startTime),
			CreatedTasks: 1,
		},
	}
)

func TestKillCommand(t *testing.T) {
	runtimetesting.RunCommandTests(t, []runtimetesting.CommandTest{
		{
			Name: "display help",
			Args: []string{"kill", "--help"},
			Stdout: runtimetesting.Output{
				Contains: cmd.KillExample,
			},
		},
		{
			Name:      "need an argument",
			Args:      []string{"kill"},
			WantError: assert.Error,
		},
		{
			Name:      "job does not exist",
			Args:      []string{"kill", "running-job"},
			WantError: testutils.AssertErrorIsNotFound(),
		},
		{
			Name: "completion only show running jobs",
			Args: []string{cobra.ShellCompRequestCmd, "kill", ""},
			Fixtures: []runtime.Object{
				jobRunning,
				jobFinished,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobRunning.Name,
				},
				ExcludesAll: []string{
					jobFinished.Name,
				},
			},
		},
		{
			Name:     "kill running job",
			Now:      testutils.Mktime(killTime),
			Args:     []string{"kill", "running-job"},
			Fixtures: []runtime.Object{runningJob},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobAction(DefaultNamespace, runningJobKilled),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches:  regexp.MustCompile(`^Requested for job [^\s]+ to be killed`),
				Contains: formatter.FormatTime(testutils.Mkmtimep(killTime)),
			},
		},
		{
			Name:      "already killed previously",
			Now:       testutils.Mktime(killTime),
			Args:      []string{"kill", "running-job"},
			Fixtures:  []runtime.Object{runningJobKilled},
			WantError: assert.Error,
		},
		{
			Name:     "kill running job with override",
			Now:      testutils.Mktime(killTime2),
			Args:     []string{"kill", "running-job", "--override"},
			Fixtures: []runtime.Object{runningJobKilled},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobAction(DefaultNamespace, runningJobKilledNewTimestamp),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches:  regexp.MustCompile(`^Requested for job [^\s]+ to be killed`),
				Contains: formatter.FormatTime(testutils.Mkmtimep(killTime2)),
			},
		},
		{
			Name:     "kill running job at specific time",
			Now:      testutils.Mktime(killTime),
			Args:     []string{"kill", "running-job", "--at", killTime2},
			Fixtures: []runtime.Object{runningJob},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobAction(DefaultNamespace, runningJobKilledNewTimestamp),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches:  regexp.MustCompile(`^Requested for job [^\s]+ to be killed`),
				Contains: formatter.FormatTime(testutils.Mkmtimep(killTime2)),
			},
		},
		{
			Name:     "kill running job after certain duration",
			Now:      testutils.Mktime(killTime),
			Args:     []string{"kill", "running-job", "--after", "40s"},
			Fixtures: []runtime.Object{runningJob},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobAction(DefaultNamespace, runningJobKilledNewTimestamp),
					},
				},
			},
			Stdout: runtimetesting.Output{
				Matches:  regexp.MustCompile(`^Requested for job [^\s]+ to be killed`),
				Contains: formatter.FormatTime(testutils.Mkmtimep(killTime2)),
			},
		},
		{
			Name:      "cannot specify both --at and --after",
			Args:      []string{"kill", "running-job", "--at", startTime, "--after", "3h"},
			Fixtures:  []runtime.Object{runningJob},
			WantError: assert.Error,
		},
		{
			Name:      "cannot specify --at in the past",
			Args:      []string{"kill", "running-job", "--at", startTime},
			Now:       testutils.Mktime(taskFinishTime),
			Fixtures:  []runtime.Object{runningJob},
			WantError: assert.Error,
		},
		{
			Name:      "cannot specify --after with negative value",
			Args:      []string{"kill", "running-job", "--after", "-15m"},
			Now:       testutils.Mktime(taskFinishTime),
			Fixtures:  []runtime.Object{runningJob},
			WantError: assert.Error,
		},
	})
}
