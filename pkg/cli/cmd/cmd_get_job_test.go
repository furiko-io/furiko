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
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	startTime      = "2021-02-09T04:02:00Z"
	taskCreateTime = "2021-02-09T04:02:01Z"
	taskLaunchTime = "2021-02-09T04:02:05Z"
	taskFinishTime = "2021-02-09T04:15:32Z"
)

var (
	jobRunning = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-running",
			Namespace: DefaultNamespace,
		},
		Status: execution.JobStatus{
			Phase: execution.JobRunning,
			Condition: execution.JobCondition{
				Running: &execution.JobConditionRunning{
					CreatedAt: testutils.Mkmtime(taskCreateTime),
					StartedAt: testutils.Mkmtime(taskLaunchTime),
				},
			},
			StartTime:    testutils.Mkmtimep(startTime),
			CreatedTasks: 1,
			Tasks: []execution.TaskRef{
				{
					Name:              "job-running.1",
					CreationTimestamp: testutils.Mkmtime(taskCreateTime),
					RunningTimestamp:  testutils.Mkmtimep(taskLaunchTime),
					Status: execution.TaskStatus{
						State: execution.TaskRunning,
					},
				},
			},
		},
	}

	jobFinished = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-finished",
			Namespace: DefaultNamespace,
		},
		Status: execution.JobStatus{
			Phase: execution.JobRetryLimitExceeded,
			Condition: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					CreatedAt:  testutils.Mkmtimep(taskCreateTime),
					StartedAt:  testutils.Mkmtimep(taskLaunchTime),
					FinishedAt: testutils.Mkmtime(taskFinishTime),
					Result:     execution.JobResultTaskFailed,
					Reason:     "Error",
					Message:    "some error message",
				},
			},
			StartTime:    testutils.Mkmtimep(startTime),
			CreatedTasks: 1,
			Tasks: []execution.TaskRef{
				{
					Name:              "job-finished.1",
					CreationTimestamp: testutils.Mkmtime(taskCreateTime),
					RunningTimestamp:  testutils.Mkmtimep(taskLaunchTime),
					FinishTimestamp:   testutils.Mkmtimep(taskFinishTime),
					Status: execution.TaskStatus{
						State:   execution.TaskFailed,
						Result:  job.GetResultPtr(execution.JobResultTaskFailed),
						Reason:  "Error",
						Message: "some error message",
					},
				},
			},
		},
	}

	jobQueued = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-queued",
			Namespace: DefaultNamespace,
		},
		Status: execution.JobStatus{
			Phase: execution.JobQueued,
			Condition: execution.JobCondition{
				Queueing: &execution.JobConditionQueueing{
					Reason:  "NotYetDue",
					Message: "Job is queued to start no earlier than " + startTime,
				},
			},
		},
	}
)

func TestGetJobCommand(t *testing.T) {
	runtimetesting.RunCommandTests(t, []runtimetesting.CommandTest{
		{
			Name: "display help",
			Args: []string{"get", "job", "--help"},
			Stdout: runtimetesting.Output{
				Contains: cmd.GetJobExample,
			},
		},
		{
			Name:      "need an argument",
			Args:      []string{"get", "job"},
			WantError: assert.Error,
		},
		{
			Name:     "get a single job",
			Args:     []string{"get", "job", "job-running", "-o", "name"},
			Fixtures: []runtime.Object{jobRunning},
			Stdout: runtimetesting.Output{
				Contains: "job.execution.furiko.io/job-running",
			},
		},
		{
			Name:     "can use alias",
			Args:     []string{"get", "jobs", "job-running", "-o", "name"},
			Fixtures: []runtime.Object{jobRunning},
			Stdout: runtimetesting.Output{
				Contains: "job.execution.furiko.io/job-running",
			},
		},
		{
			Name:     "get a single job, pretty print",
			Args:     []string{"get", "job", "job-running"},
			Fixtures: []runtime.Object{jobRunning},
			Stdout: runtimetesting.Output{
				// We expect some important information to be printed, such as phase, start
				// time, etc.
				ContainsAll: []string{
					string(jobRunning.Status.Phase),
					formatter.FormatTimeWithTimeAgo(testutils.Mkmtimep(taskCreateTime)),
					formatter.FormatTimeWithTimeAgo(testutils.Mkmtimep(taskLaunchTime)),
				},
			},
		},
		{
			Name:     "get a finished job, pretty print",
			Args:     []string{"get", "job", "job-finished"},
			Fixtures: []runtime.Object{jobFinished},
			Stdout: runtimetesting.Output{
				// We expect some important information to be printed, such as phase, start
				// time, etc.
				ContainsAll: []string{
					jobFinished.GetName(),
					string(jobFinished.Status.Phase),
					formatter.FormatTimeWithTimeAgo(testutils.Mkmtimep(taskCreateTime)),
					formatter.FormatTimeWithTimeAgo(testutils.Mkmtimep(taskLaunchTime)),
					formatter.FormatTimeWithTimeAgo(testutils.Mkmtimep(taskFinishTime)),
					"some error message",
				},
			},
		},
		{
			Name:      "get job does not exist",
			Args:      []string{"get", "job", "job-running"},
			WantError: runtimetesting.AssertErrorIsNotFound(),
		},
	})
}
