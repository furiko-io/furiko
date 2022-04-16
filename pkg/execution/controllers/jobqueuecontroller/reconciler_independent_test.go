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

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobqueuecontroller"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

func TestIndependentReconciler(t *testing.T) {
	test := runtimetesting.ReconcilerTest{
		ReconcilerFunc: func(c controllercontext.Context) (reconciler.Reconciler, []cache.InformerSynced) {
			ctrlCtx := jobqueuecontroller.NewContextWithRecorder(c, &record.FakeRecorder{})
			recon := jobqueuecontroller.NewIndependentReconciler(ctrlCtx, &configv1alpha1.Concurrency{
				Workers: 1,
			})
			return recon, ctrlCtx.HasSynced
		},
		Now: testutils.Mktime(now),
	}

	test.Run(t, []runtimetesting.ReconcilerTestCase{
		{
			Name:   "start job",
			Target: jobToBeStarted,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace, startJob(jobToBeStarted, timeNow)),
					},
				},
			},
		},
		{
			Name:   "job already started",
			Target: startJob(jobToBeStarted, timeNow),
		},
		{
			Name:   "don't start job with future startAfter",
			Target: jobWithStartAfter,
		},
		{
			Name:   "start job with past startAfter",
			Now:    testutils.Mktime(startAfter),
			Target: jobWithStartAfter,
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobStatusAction(jobNamespace,
							startJob(jobWithStartAfter, testutils.Mkmtimep(startAfter))),
					},
				},
			},
		},
	})
}
