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

package croncontroller_test

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/croncontroller"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	utilatomic "github.com/furiko-io/furiko/pkg/utils/atomic"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

var (
	scheduleSpecEvery5Min = &execution.ScheduleSpec{
		Cron: &execution.CronSchedule{
			Expression: "0/5 * * * *",
		},
	}

	jobConfigForbid = makeJobConfig("job-config-forbid", execution.JobConfigSpec{
		Schedule: scheduleSpecEvery5Min,
		Concurrency: execution.ConcurrencySpec{
			Policy: execution.ConcurrencyPolicyForbid,
		},
	})

	jobConfigAllow = makeJobConfig("job-config-allow", execution.JobConfigSpec{
		Schedule: scheduleSpecEvery5Min,
		Concurrency: execution.ConcurrencySpec{
			Policy: execution.ConcurrencyPolicyAllow,
		},
	})
)

func TestReconciler(t *testing.T) {
	type syncTarget struct {
		namespace string
		name      string
	}
	tests := []struct {
		name              string
		syncTarget        syncTarget
		initialJobConfigs []*execution.JobConfig
		initialCounts     map[*execution.JobConfig]int64
		control           MockControl
		wantNumCreated    int
		wantErr           bool
	}{
		{
			name: "cannot split name",
			syncTarget: syncTarget{
				namespace: testNamespace,
				name:      "",
			},
			wantErr: true,
		},
		{
			name: "no such job config, don't throw error",
			syncTarget: syncTarget{
				namespace: testNamespace,
				name:      croncontroller.JoinJobConfigKeyName(testName, testutils.Mktime(scheduleTime)),
			},
		},
		{
			name: "create job successfully",
			initialJobConfigs: []*execution.JobConfig{
				jobConfigForbid,
				jobConfigAllow,
			},
			syncTarget: syncTarget{
				namespace: jobConfigAllow.Namespace,
				name:      croncontroller.JoinJobConfigKeyName(jobConfigAllow.Name, testutils.Mktime(scheduleTime)),
			},
			wantNumCreated: 1,
		},
		{
			name: "failed to create job",
			initialJobConfigs: []*execution.JobConfig{
				jobConfigForbid,
				jobConfigAllow,
			},
			control: newMockControlWithError(errors.New("error")),
			syncTarget: syncTarget{
				namespace: jobConfigAllow.Namespace,
				name:      croncontroller.JoinJobConfigKeyName(jobConfigAllow.Name, testutils.Mktime(scheduleTime)),
			},
			wantErr: true,
		},
		{
			name: "create job for Forbid",
			initialJobConfigs: []*execution.JobConfig{
				jobConfigForbid,
				jobConfigAllow,
			},
			syncTarget: syncTarget{
				namespace: jobConfigForbid.Namespace,
				name:      croncontroller.JoinJobConfigKeyName(jobConfigForbid.Name, testutils.Mktime(scheduleTime)),
			},
			wantNumCreated: 1,
		},
		{
			name: "create job for Allow with active",
			initialJobConfigs: []*execution.JobConfig{
				jobConfigForbid,
				jobConfigAllow,
			},
			initialCounts: map[*execution.JobConfig]int64{
				jobConfigAllow:  1,
				jobConfigForbid: 1,
			},
			syncTarget: syncTarget{
				namespace: jobConfigAllow.Namespace,
				name:      croncontroller.JoinJobConfigKeyName(jobConfigAllow.Name, testutils.Mktime(scheduleTime)),
			},
			wantNumCreated: 1,
		},
		{
			name: "do not create job for Forbid with active",
			initialJobConfigs: []*execution.JobConfig{
				jobConfigForbid,
				jobConfigAllow,
			},
			initialCounts: map[*execution.JobConfig]int64{
				jobConfigAllow:  1,
				jobConfigForbid: 1,
			},
			syncTarget: syncTarget{
				namespace: jobConfigForbid.Namespace,
				name:      croncontroller.JoinJobConfigKeyName(jobConfigForbid.Name, testutils.Mktime(scheduleTime)),
			},
		},
		{
			name: "can create job for Forbid with other job active",
			initialJobConfigs: []*execution.JobConfig{
				jobConfigForbid,
				jobConfigAllow,
			},
			initialCounts: map[*execution.JobConfig]int64{
				jobConfigAllow: 1,
			},
			syncTarget: syncTarget{
				namespace: jobConfigForbid.Namespace,
				name:      croncontroller.JoinJobConfigKeyName(jobConfigForbid.Name, testutils.Mktime(scheduleTime)),
			},
			wantNumCreated: 1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			c := mock.NewContext()
			ctrlCtx := croncontroller.NewContext(c)
			control := tt.control
			if control == nil {
				control = &mockControl{}
			}
			store := newMockStore(tt.initialCounts)
			reconciler := croncontroller.NewReconciler(ctrlCtx, control, store, runtimetesting.ReconcilerDefaultConcurrency)

			err := c.Start(ctx)
			assert.NoError(t, err)
			client := c.MockClientsets()

			// Create initial objects
			for _, rjc := range tt.initialJobConfigs {
				rjc := rjc.DeepCopy()
				_, err := client.Furiko().ExecutionV1alpha1().JobConfigs(rjc.Namespace).Create(ctx, rjc, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("cannot create JobConfig: %v", err)
				}
			}

			// NOTE(irvinlim): Add a short delay otherwise cache may not sync consistently
			time.Sleep(time.Millisecond * 10)

			// Wait for cache sync
			if !cache.WaitForCacheSync(ctx.Done(), ctrlCtx.HasSynced...) {
				assert.FailNow(t, "caches not synced")
			}

			// Trigger sync
			if err := reconciler.SyncOne(ctx, tt.syncTarget.namespace, tt.syncTarget.name, 0); (err != nil) != tt.wantErr {
				t.Errorf("SyncOne() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Assert call count.
			assert.Equal(t, control.CountCreatedJobs(), tt.wantNumCreated)
		})
	}
}

type MockControl interface {
	croncontroller.ExecutionControlInterface
	CountCreatedJobs() int
}

type mockControl struct {
	createdJobs []*execution.Job
}

func newMockControl() *mockControl {
	return &mockControl{}
}

var _ MockControl = (*mockControl)(nil)

func (m *mockControl) CountCreatedJobs() int {
	return len(m.createdJobs)
}

func (m *mockControl) CreateJob(_ context.Context, _ *execution.JobConfig, rj *execution.Job) error {
	m.createdJobs = append(m.createdJobs, rj)
	return nil
}

type mockControlWithError struct {
	*mockControl
	error error
}

func newMockControlWithError(error error) *mockControlWithError {
	return &mockControlWithError{
		mockControl: newMockControl(),
		error:       error,
	}
}

func (m *mockControlWithError) CreateJob(_ context.Context, _ *execution.JobConfig, rj *execution.Job) error {
	return m.error
}

type mockStore struct {
	*utilatomic.Counter
}

func newMockStore(initial map[*execution.JobConfig]int64) *mockStore {
	counter := utilatomic.NewCounter()
	for rjc, count := range initial {
		for i := int64(0); i < count; i++ {
			counter.Add(string(rjc.UID))
		}
	}
	return &mockStore{
		Counter: counter,
	}
}

var _ controllercontext.ActiveJobStore = (*mockStore)(nil)

func (m *mockStore) CountActiveJobsForConfig(rjc *execution.JobConfig) int64 {
	return m.Counter.Get(string(rjc.UID))
}

func (m *mockStore) CheckAndAdd(rjc *execution.JobConfig, oldCount int64) (success bool) {
	return m.Counter.CheckAndAdd(string(rjc.UID), oldCount)
}

func (m *mockStore) Delete(rjc *execution.JobConfig) {
	m.Counter.Remove(string(rjc.UID))
}

// makeJobConfig returns a new JobConfig with the given Spec.
func makeJobConfig(name string, spec execution.JobConfigSpec) *execution.JobConfig {
	return &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      name,
			UID:       testutils.MakeUID(name),
		},
		Spec: spec,
	}
}
