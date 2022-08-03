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
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/cmd"
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/execution/util/parallel"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/errors"
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
					LatestCreationTimestamp: testutils.Mkmtime(taskCreateTime),
					LatestRunningTimestamp:  testutils.Mkmtime(taskLaunchTime),
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
			Phase: execution.JobFailed,
			Condition: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					LatestCreationTimestamp: testutils.Mkmtimep(taskCreateTime),
					LatestRunningTimestamp:  testutils.Mkmtimep(taskLaunchTime),
					FinishTimestamp:         testutils.Mkmtime(taskFinishTime),
					Result:                  execution.JobResultFailed,
					Reason:                  "Error",
					Message:                 "some error message",
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
						State:   execution.TaskTerminated,
						Result:  execution.TaskFailed,
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

	jobParallel = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-parallel",
			Namespace: DefaultNamespace,
		},
		Spec: execution.JobSpec{
			Template: &execution.JobTemplate{
				Parallelism: &execution.ParallelismSpec{
					WithCount:          pointer.Int64(3),
					CompletionStrategy: execution.AllSuccessful,
				},
			},
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
			CreatedTasks: 3,
			ParallelStatus: &execution.ParallelStatus{
				ParallelStatusSummary: execution.ParallelStatusSummary{
					Complete: false,
				},
				Indexes: []execution.ParallelIndexStatus{
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(0),
						},
						Hash:         makeHash(0),
						CreatedTasks: 1,
						State:        execution.IndexRunning,
					},
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(1),
						},
						Hash:         makeHash(1),
						CreatedTasks: 1,
						State:        execution.IndexRunning,
					},
					{
						Index: execution.ParallelIndex{
							IndexNumber: pointer.Int64(2),
						},
						Hash:         makeHash(2),
						CreatedTasks: 1,
						State:        execution.IndexRunning,
					},
				},
			},
			Tasks: []execution.TaskRef{
				{
					Name:              "job-parallel-0",
					CreationTimestamp: testutils.Mkmtime(taskCreateTime),
					RunningTimestamp:  testutils.Mkmtimep(taskLaunchTime),
					Status: execution.TaskStatus{
						State: execution.TaskRunning,
					},
				},
				{
					Name:              "job-parallel-0",
					CreationTimestamp: testutils.Mkmtime(taskCreateTime),
					RunningTimestamp:  testutils.Mkmtimep(taskLaunchTime),
					Status: execution.TaskStatus{
						State: execution.TaskRunning,
					},
				},
				{
					Name:              "job-parallel-0",
					CreationTimestamp: testutils.Mkmtime(taskCreateTime),
					RunningTimestamp:  testutils.Mkmtimep(taskLaunchTime),
					Status: execution.TaskStatus{
						State: execution.TaskRunning,
					},
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
			Name:     "get a finished job, pretty print with detail",
			Args:     []string{"get", "job", "job-finished", "-o", "detail"},
			Fixtures: []runtime.Object{jobFinished},
			Stdout: runtimetesting.Output{
				// Should print list of tasks and their details.
				ContainsAll: []string{
					jobFinished.Status.Tasks[0].Name,
					jobFinished.Status.Tasks[0].Status.Message,
				},
			},
		},
		{
			Name:     "get a parallel job",
			Args:     []string{"get", "job", "job-parallel"},
			Fixtures: []runtime.Object{jobParallel},
			Stdout: runtimetesting.Output{
				// Should print parallel task summary.
				ContainsAll: []string{
					string(jobParallel.Spec.Template.Parallelism.GetCompletionStrategy()),
					"3 Running",
				},
			},
		},
		{
			Name:     "get a parallel job with detail",
			Args:     []string{"get", "job", "job-parallel", "-o", "detail"},
			Fixtures: []runtime.Object{jobParallel},
			Stdout: runtimetesting.Output{
				// Should print parallel task groups.
				ContainsAll: []string{
					makeHash(0),
				},
				MatchesAll: []*regexp.Regexp{
					regexp.MustCompile(`Index Number:\s+0`),
				},
			},
		},
		{
			Name:      "get job does not exist",
			Args:      []string{"get", "job", "job-running"},
			WantError: errors.AssertIsNotFound(),
		},
	})
}

func makeHash(index int64) string {
	hash, err := parallel.HashIndex(execution.ParallelIndex{
		IndexNumber: pointer.Int64(index),
	})
	if err != nil {
		panic(err)
	}
	return hash
}
