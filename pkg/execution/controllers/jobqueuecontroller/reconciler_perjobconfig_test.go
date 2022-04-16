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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobqueuecontroller"
	"github.com/furiko-io/furiko/pkg/execution/stores/activejobstore"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

func TestPerJobConfigReconciler(t *testing.T) {
	tests := []struct {
		name             string
		target           *execution.JobConfig
		targetGenerator  func() *execution.JobConfig
		initialJobs      []*execution.Job
		wantErr          bool
		now              time.Time
		executionActions runtimetesting.ActionTest
		configs          map[configv1alpha1.ConfigName]runtime.Object
	}{
		{
			name:   "job config with no jobs",
			target: jobConfig1,
		},
		{
			name:   "job config with unstarted job",
			target: jobConfig1,
			initialJobs: []*execution.Job{
				jobForConfig1ToBeStarted,
			},
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateStatusAction(resourceJob, jobConfig1.Namespace,
						startJob(jobForConfig1ToBeStarted, timeNow)),
				},
			},
		},
		{
			name:   "job config with already started job",
			target: jobConfig1,
			initialJobs: []*execution.Job{
				startJob(jobForConfig1ToBeStarted, timeNow),
			},
		},
		{
			name:   "job config with unstarted job for another job config",
			target: jobConfig1,
			initialJobs: []*execution.Job{
				jobForConfig2ToBeStarted,
			},
		},
		{
			name:   "job config with unstarted independent job",
			target: jobConfig1,
			initialJobs: []*execution.Job{
				jobToBeStarted,
			},
		},
		{
			name: "don't start job with future startAfter",
			initialJobs: []*execution.Job{
				jobForConfig1WithStartAfter,
			},
			target: jobConfig1,
		},
		{
			name: "start job with past startAfter",
			now:  testutils.Mktime(startAfter),
			initialJobs: []*execution.Job{
				jobForConfig1WithStartAfter,
			},
			target: jobConfig1,
			executionActions: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateStatusAction(resourceJob, jobNamespace,
						startJob(jobForConfig1WithStartAfter, testutils.Mkmtimep(startAfter))),
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			c := mock.NewContext()
			ctrlCtx := jobqueuecontroller.NewContextWithRecorder(c, &record.FakeRecorder{})
			reconciler := jobqueuecontroller.NewPerConfigReconciler(ctrlCtx, &configv1alpha1.Concurrency{
				Workers: 1,
			})
			c.MockConfigs().SetConfigs(tt.configs)
			c.MockStores().RegisterFromFactoriesOrDie(activejobstore.NewFactory())

			err := c.Start(ctx)
			assert.NoError(t, err)
			client := c.MockClientsets()

			// Set the current time
			fakeNow := testutils.Mktime(now)
			if !tt.now.IsZero() {
				fakeNow = tt.now
			}
			fakeClock := clock.NewFakeClock(fakeNow)
			ktime.Clock = fakeClock

			// Create initial objects
			for _, job := range tt.initialJobs {
				job := job.DeepCopy()
				_, err := client.Furiko().ExecutionV1alpha1().Jobs(job.Namespace).
					Create(ctx, job, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("cannot create Job: %v", err)
				}
			}

			// Create object
			target := tt.target
			if tt.targetGenerator != nil {
				target = tt.targetGenerator()
			}
			target = target.DeepCopy()
			created, err := client.Furiko().ExecutionV1alpha1().JobConfigs(target.Namespace).
				Create(ctx, target, metav1.CreateOptions{})
			assert.NoError(t, err)

			// NOTE(irvinlim): Add a short delay otherwise cache may not sync consistently
			time.Sleep(time.Millisecond * 10)

			// Wait for cache sync
			if !cache.WaitForCacheSync(ctx.Done(), ctrlCtx.HasSynced...) {
				assert.FailNow(t, "caches not synced")
			}

			// Clear all actions prior to reconcile
			client.ClearActions()

			// Reconcile object
			if err := reconciler.SyncOne(ctx, created.Namespace, created.Name, 0); (err != nil) != tt.wantErr {
				t.Errorf("SyncOne() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Compare actions.
			runtimetesting.CompareActions(t, tt.executionActions, client.FurikoMock().Actions())
		})
	}
}
