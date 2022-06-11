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

package jobqueuecontroller_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobqueuecontroller"
	"github.com/furiko-io/furiko/pkg/execution/stores/activejobstore"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

func TestPerJobConfigReconciler(t *testing.T) {
	test := runtimetesting.ReconcilerTest{
		ContextFunc: func(c controllercontext.Context, recorder record.EventRecorder) runtimetesting.ControllerContext {
			return jobqueuecontroller.NewContextWithRecorder(c, recorder)
		},
		ReconcilerFunc: func(c runtimetesting.ControllerContext) reconciler.Reconciler {
			return jobqueuecontroller.NewPerConfigReconciler(
				c.(*jobqueuecontroller.Context),
				runtimetesting.ReconcilerDefaultConcurrency,
				newMockJobControl(c.Clientsets().Furiko().ExecutionV1alpha1()),
			)
		},
		Now: testutils.Mktime(now),
		Stores: []mock.StoreFactory{
			activejobstore.NewFactory(),
		},
	}

	test.Run(t, []runtimetesting.ReconcilerTestCase{
		{
			Name:   "job config with no jobs",
			Target: jobConfig1,
		},
		{
			Name:   "job config with unstarted job",
			Target: jobConfig1,
			Fixtures: []runtime.Object{
				jobForConfig1ToBeStarted,
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobConfig1.Namespace,
							startJob(jobForConfig1ToBeStarted, timeNow)),
					},
				},
			},
		},
		{
			Name:   "job config with already started job",
			Target: jobConfig1,
			Fixtures: []runtime.Object{
				startJob(jobForConfig1ToBeStarted, timeNow),
			},
		},
		{
			Name:   "job config with unstarted job for another job config",
			Target: jobConfig1,
			Fixtures: []runtime.Object{
				jobForConfig2ToBeStarted,
			},
		},
		{
			Name:   "job config with unstarted independent job",
			Target: jobConfig1,
			Fixtures: []runtime.Object{
				jobToBeStarted,
			},
		},
		{
			Name: "don't start job with future startAfter",
			Fixtures: []runtime.Object{
				jobForConfig1WithStartAfter,
			},
			Target: jobConfig1,
		},
		{
			Name: "start job with past startAfter",
			Now:  testutils.Mktime(startAfter),
			Fixtures: []runtime.Object{
				jobForConfig1WithStartAfter,
			},
			Target: jobConfig1,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace,
							startJob(jobForConfig1WithStartAfter, testutils.Mkmtimep(startAfter))),
					},
				},
			},
		},
		{
			Name: "reject second job with Forbid",
			Fixtures: []runtime.Object{
				jobForForbid1,
				jobForForbid2,
			},
			Target: jobConfigForbid,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, startJob(jobForForbid1, timeNow)),
						runtimetesting.NewUpdateJobAction(jobNamespace, rejectJob(jobForForbid2)),
					},
				},
			},
		},
		{
			Name: "wait for second job with Enqueue",
			Fixtures: []runtime.Object{
				jobForForbid1,
				makeJob("job2", 1, jobConfigForbid, execution.ConcurrencyPolicyEnqueue, nil),
			},
			Target: jobConfigForbid,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, startJob(jobForForbid1, timeNow)),
					},
				},
			},
		},
		{
			Name: "will skip first job with StartAfter",
			Fixtures: []runtime.Object{
				makeJob("job1", 0, jobConfigForbid, execution.ConcurrencyPolicyForbid, testutils.Mkmtimep(startAfter)),
				jobForForbid2,
			},
			Target: jobConfigForbid,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, startJob(jobForForbid2, timeNow)),
					},
				},
			},
		},
		{
			Name: "can start both jobs with MaxConcurrency set to 2",
			Fixtures: []runtime.Object{
				jobForForbid1,
				jobForForbid2,
			},
			Target: jobConfigForbidMax2,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, startJob(jobForForbid1, timeNow)),
						runtimetesting.NewUpdateJobAction(jobNamespace, startJob(jobForForbid2, timeNow)),
					},
				},
			},
		},
		{
			Name: "reject 3rd job with MaxConcurrency set to 2",
			Fixtures: []runtime.Object{
				jobForForbid1,
				jobForForbid2,
				makeJob("job3", 2, jobConfigForbid, execution.ConcurrencyPolicyForbid, nil),
			},
			Target: jobConfigForbidMax2,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, startJob(jobForForbid1, timeNow)),
						runtimetesting.NewUpdateJobAction(jobNamespace, startJob(jobForForbid2, timeNow)),
						runtimetesting.NewUpdateJobAction(jobNamespace, rejectJob(
							makeJob("job3", 2, jobConfigForbid, execution.ConcurrencyPolicyForbid, nil),
						)),
					},
				},
			},
		},
	})
}
