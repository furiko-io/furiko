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

package completion_test

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

var (
	defaultJobRunning = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "job-running",
			Namespace:         DefaultNamespace,
			CreationTimestamp: testutils.Mkmtime(executeTime),
		},
		Status: execution.JobStatus{
			Phase:        execution.JobRunning,
			CreatedTasks: 1,
			RunningTasks: 1,
			StartTime:    testutils.Mkmtimep(executeTime),
		},
	}

	defaultJobFinished = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "job-finished",
			Namespace:         DefaultNamespace,
			CreationTimestamp: testutils.Mkmtime(executeTime2),
		},
		Status: execution.JobStatus{
			Phase:        execution.JobSucceeded,
			CreatedTasks: 2,
			StartTime:    testutils.Mkmtimep(executeTime2),
			Condition: execution.JobCondition{
				Finished: &execution.JobConditionFinished{
					FinishTimestamp: testutils.Mkmtime(finishTime2),
				},
			},
		},
	}

	defaultJobQueued = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "job-queued",
			Namespace:         DefaultNamespace,
			CreationTimestamp: testutils.Mkmtime(executeTime3),
		},
		Status: execution.JobStatus{
			Phase: execution.JobQueued,
		},
	}

	prodJob = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "prod-job",
			Namespace:         ProdNamespace,
			CreationTimestamp: testutils.Mkmtime(executeTime),
		},
		Status: execution.JobStatus{
			Phase:     execution.JobRunning,
			StartTime: testutils.Mkmtimep(executeTime),
		},
	}
)

func TestListJobsCompleter(t *testing.T) {
	runtimetesting.RunCompleterTests(t, []runtimetesting.CompleterTest{
		{
			Name:      "no jobs",
			Completer: &completion.ListJobsCompleter{},
			Want:      []string{},
		},
		{
			Name: "show single job",
			Fixtures: []runtime.Object{
				defaultJobRunning,
			},
			Now:       testutils.Mktime(currentTime),
			Completer: &completion.ListJobsCompleter{},
			Want: []string{
				`job-running	Running, started Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
			},
		},
		{
			Name: "show creation timestamp for non-started job",
			Fixtures: []runtime.Object{
				defaultJobQueued,
			},
			Now:       testutils.Mktime(currentTime),
			Completer: &completion.ListJobsCompleter{},
			Want: []string{
				`job-queued	Queued, created Tue, 09 Feb 2021 05:02:00 +00:00 (5 hours ago)`,
			},
		},
		{
			Name: "show default namespace",
			Fixtures: []runtime.Object{
				defaultJobRunning,
				prodJob,
			},
			Namespace: DefaultNamespace,
			Now:       testutils.Mktime(currentTime),
			Completer: &completion.ListJobsCompleter{},
			Want: []string{
				`job-running	Running, started Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
			},
		},
		{
			Name: "show prod namespace",
			Fixtures: []runtime.Object{
				defaultJobRunning,
				prodJob,
			},
			Namespace: ProdNamespace,
			Now:       testutils.Mktime(currentTime),
			Completer: &completion.ListJobsCompleter{},
			Want: []string{
				`prod-job	Running, started Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
			},
		},
		{
			Name: "sort by creation timestamp",
			Fixtures: []runtime.Object{
				defaultJobRunning,
				defaultJobFinished,
				defaultJobQueued,
			},
			Now:       testutils.Mktime(currentTime),
			Completer: &completion.ListJobsCompleter{},
			Want: []string{
				`job-queued	Queued, created Tue, 09 Feb 2021 05:02:00 +00:00 (5 hours ago)`,
				`job-running	Running, started Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
				`job-finished	Succeeded, finished Tue, 09 Feb 2021 03:47:00 +00:00 (6 hours ago)`,
			},
		},
		{
			Name: "with sort",
			Fixtures: []runtime.Object{
				defaultJobRunning,
				defaultJobFinished,
				defaultJobQueued,
			},
			Now: testutils.Mktime(currentTime),
			Completer: &completion.ListJobsCompleter{
				Sort: func(a, b *execution.Job) bool {
					return a.Name < b.Name
				},
			},
			Want: []string{
				`job-finished	Succeeded, finished Tue, 09 Feb 2021 03:47:00 +00:00 (6 hours ago)`,
				`job-queued	Queued, created Tue, 09 Feb 2021 05:02:00 +00:00 (5 hours ago)`,
				`job-running	Running, started Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
			},
		},
		{
			Name: "with filter",
			Fixtures: []runtime.Object{
				defaultJobRunning,
				defaultJobFinished,
				defaultJobQueued,
			},
			Now: testutils.Mktime(currentTime),
			Completer: &completion.ListJobsCompleter{
				Filter: jobutil.IsActive,
			},
			Want: []string{
				`job-running	Running, started Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
			},
		},
		{
			Name: "with format",
			Fixtures: []runtime.Object{
				defaultJobRunning,
				defaultJobFinished,
				defaultJobQueued,
			},
			Completer: &completion.ListJobsCompleter{
				Format: func(job *execution.Job) string {
					var label string
					if job.Status.CreatedTasks > 0 {
						label = fmt.Sprintf("%v/%v tasks running", job.Status.RunningTasks, job.Status.CreatedTasks)
					}
					return fmt.Sprintf("%v\t%v", job.Name, label)
				},
			},
			Want: []string{
				`job-queued	`,
				`job-running	1/1 tasks running`,
				`job-finished	0/2 tasks running`,
			},
		},
	})
}
