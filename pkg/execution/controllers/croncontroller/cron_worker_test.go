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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/croncontroller"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

var (
	cronWorkerJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-test",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "* * * * *",
				},
			},
		},
	}

	cronWorkerJobConfigUpdateToEvery15Sec = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-test",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "0/15 * * * * * *",
				},
			},
		},
	}

	cronWorkerJobConfigDaily = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-daily",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "0 10 * * *",
				},
			},
		},
	}

	cronWorkerJobConfigDailySingapore = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-config-daily-singapore",
		},
		Spec: execution.JobConfigSpec{
			Schedule: &execution.ScheduleSpec{
				Cron: &execution.CronSchedule{
					Expression: "0 10 * * *",
					Timezone:   "Asia/Singapore",
				},
			},
		},
	}
)

func TestCronWorker(t *testing.T) { // nolint:gocognit
	type step struct {
		Name        string
		Time        time.Time
		Update      *execution.JobConfig
		Delete      *execution.JobConfig
		WantEnqueue []string
	}
	tests := []struct {
		name       string
		jobConfigs []*execution.JobConfig
		configs    map[configv1alpha1.ConfigName]runtime.Object
		steps      []step
	}{
		{
			name: "Scheduled every minute",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfig,
			},
			steps: []step{
				{
					Name: "Initial time",
					Time: testutils.Mktime("2022-04-01T10:52:04Z"),
				},
				{
					Name: "No enqueue until end of minute",
					Time: testutils.Mktime("2022-04-01T10:52:59Z"),
				},
				{
					Name: "Want enqueue at start of minute",
					Time: testutils.Mktime("2022-04-01T10:53:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:53:00Z")),
					},
				},
				{
					Name: "No more enqueue at next second",
					Time: testutils.Mktime("2022-04-01T10:53:01Z"),
				},
				{
					Name: "Want enqueue at start of next minute",
					Time: testutils.Mktime("2022-04-01T10:54:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:54:00Z")),
					},
				},
				{
					Name: "Multiple enqueue when jumping a few minutes",
					Time: testutils.Mktime("2022-04-01T10:57:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:55:00Z")),
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:56:00Z")),
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:57:00Z")),
					},
				},
			},
		},
		{
			name: "Update JobConfig",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfig,
			},
			steps: []step{
				{
					Name: "Initial time",
					Time: testutils.Mktime("2022-04-01T10:52:04Z"),
				},
				{
					Name: "Enqueue at next minute",
					Time: testutils.Mktime("2022-04-01T10:53:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:53:00Z")),
					},
				},
				{
					Name: "Update time",
					Time: testutils.Mktime("2022-04-01T10:53:10Z"),
				},
				{
					Name:   "Update JobConfig",
					Update: cronWorkerJobConfigUpdateToEvery15Sec,
				},
				{
					Name: "No enqueue yet",
					Time: testutils.Mktime("2022-04-01T10:53:11Z"),
				},
				{
					Name: "Want enqueue at 15 sec mark",
					Time: testutils.Mktime("2022-04-01T10:53:15Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:53:15Z")),
					},
				},
			},
		},
		{
			name: "Delete JobConfig",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfig,
			},
			steps: []step{
				{
					Name: "Initial time",
					Time: testutils.Mktime("2022-04-01T10:52:04Z"),
				},
				{
					Name: "Enqueue at next minute",
					Time: testutils.Mktime("2022-04-01T10:53:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:53:00Z")),
					},
				},
				{
					Name:   "Delete JobConfig",
					Delete: cronWorkerJobConfig,
				},
				{
					Name: "No enqueue 1 minute later",
					Time: testutils.Mktime("2022-04-01T10:54:00Z"),
				},
				{
					Name: "No enqueue 2 minutes later",
					Time: testutils.Mktime("2022-04-01T10:55:00Z"),
				},
			},
		},
		{
			name: "Scheduled daily",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfigDaily,
			},
			steps: []step{
				{
					Name: "Initial time",
					Time: testutils.Mktime("2022-04-01T10:52:04Z"),
				},
				{
					Name: "Enqueue next day",
					Time: testutils.Mktime("2022-04-02T10:00:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfigDaily, testutils.Mktime("2022-04-02T10:00:00Z")),
					},
				},
			},
		},
		{
			name: "Scheduled daily with default configured timezone",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfigDaily,
			},
			configs: map[configv1alpha1.ConfigName]runtime.Object{
				configv1alpha1.CronExecutionConfigName: &configv1alpha1.CronExecutionConfig{
					DefaultTimezone: pointer.String("America/New_York"),
				},
			},
			steps: []step{
				{
					Name: "Initial time",
					Time: testutils.Mktime("2022-04-01T10:52:04Z"),
				},
				{
					Name: "Enqueue at 14:00 UTC",
					Time: testutils.Mktime("2022-04-01T14:00:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfigDaily, testutils.Mktime("2022-04-01T14:00:00Z")),
					},
				},
			},
		},
		{
			name: "Scheduled daily with job configured timezone",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfigDailySingapore,
			},
			configs: map[configv1alpha1.ConfigName]runtime.Object{
				configv1alpha1.CronExecutionConfigName: &configv1alpha1.CronExecutionConfig{
					DefaultTimezone: pointer.String("America/New_York"),
				},
			},
			steps: []step{
				{
					Name: "Initial time",
					Time: testutils.Mktime("2022-04-01T10:52:04Z"),
				},
				{
					Name: "No enqueue at 14:00 UTC",
					Time: testutils.Mktime("2022-04-01T14:00:00Z"),
				},
				{
					Name: "Enqueue at 02:00 UTC next day",
					Time: testutils.Mktime("2022-04-02T02:00:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfigDailySingapore, testutils.Mktime("2022-04-02T02:00:00Z")),
					},
				},
			},
		},
		{
			name: "Missed too many schedules",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfig,
			},
			configs: map[configv1alpha1.ConfigName]runtime.Object{
				configv1alpha1.CronExecutionConfigName: &configv1alpha1.CronExecutionConfig{
					MaxMissedSchedules: pointer.Int64(3),
				},
			},
			steps: []step{
				{
					Name: "Initial time",
					Time: testutils.Mktime("2022-04-01T10:52:04Z"),
				},
				{
					Name: "Want enqueue at start of minute",
					Time: testutils.Mktime("2022-04-01T10:53:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:53:00Z")),
					},
				},
				{
					Name: "Enqueue only 3 jobs",
					Time: testutils.Mktime("2022-04-01T10:59:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:54:00Z")),
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:55:00Z")),
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:56:00Z")),
					},
				},
				{
					Name: "Enqueue once for next minute",
					Time: testutils.Mktime("2022-04-01T11:00:00Z"),
					WantEnqueue: []string{
						keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T11:00:00Z")),
					},
				},
			},
		},
		{
			name: "No future schedules",
			jobConfigs: []*execution.JobConfig{
				jobConfigPointInTime,
			},
			steps: []step{
				{
					Name: "Initial time",
					Time: testutils.Mktime("2021-02-09T10:52:04Z"),
				},
				{
					Name: "Enqueue job",
					Time: testutils.Mktime("2021-02-09T12:00:00Z"),
					WantEnqueue: []string{
						keyFunc(jobConfigPointInTime, testutils.Mktime("2021-02-09T12:00:00Z")),
					},
				},
				{
					Name: "No more future schedules",
					Time: testutils.Mktime("2099-12-31T23:59:59Z"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var prevtime time.Time
			fakeClock := clock.NewFakeClock(prevtime)
			croncontroller.Clock = fakeClock

			// Initialize contexts and worker
			c := mock.NewContext()
			c.MockConfigs().SetConfigs(tt.configs)
			ctrlContext := croncontroller.NewContext(c)
			queue := newEnqueueHandler()
			worker := croncontroller.NewCronWorker(ctrlContext, queue)
			executionClient := c.MockClientsets().Furiko().ExecutionV1alpha1()

			// Use custom handler to intercept updates.
			handler := newNotifyingUpdateHandler(croncontroller.NewUpdateHandler(ctrlContext))

			// Initialize InformerWorker, so that we can receive events from clientset
			// updates.
			informer := croncontroller.NewInformerWorker(ctrlContext, handler)
			informer.Init()

			// Start context.
			assert.NoError(t, c.Start(ctx))

			// Initialize fixtures
			for _, jobConfig := range tt.jobConfigs {
				_, err := executionClient.JobConfigs(jobConfig.Namespace).Create(ctx, jobConfig, metav1.CreateOptions{})
				assert.NoError(t, err)
			}

			// Wait for cache sync
			if !cache.WaitForCacheSync(ctx.Done(), ctrlContext.HasSynced...) {
				assert.FailNow(t, "caches not synced")
			}

			// Start CronWorker in background. This is mainly a no-op because we will be
			// triggering Work manually in each step below.
			worker.Start(ctx)

			for _, step := range tt.steps {
				msgAndArgs := []interface{}{
					fmt.Sprintf("Error in step: %v", step.Name),
				}

				// Update clock, ensure to never move backwards in time.
				if !step.Time.IsZero() {
					if prevtime.After(step.Time) {
						t.Errorf("cannot set now to %v, previous time was %v", step.Time, prevtime)
						return
					}
					prevtime = step.Time
					fakeClock.SetTime(step.Time)
				}

				// Perform update step.
				if jobConfig := step.Update; jobConfig != nil {
					_, err := executionClient.JobConfigs(jobConfig.Namespace).
						Update(ctx, jobConfig, metav1.UpdateOptions{})
					assert.NoError(t, err)
					handler.Wait()
				}

				// Perform delete step.
				if jobConfig := step.Delete; jobConfig != nil {
					err := executionClient.JobConfigs(jobConfig.Namespace).
						Delete(ctx, jobConfig.Name, metav1.DeleteOptions{})
					assert.NoError(t, err)
					handler.Wait()
				}

				// Trigger work manually.
				worker.Work()

				// Consume from queue.
				for _, key := range step.WantEnqueue {
					item, ok := queue.Get()
					assert.True(t, ok, msgAndArgs...)
					assert.Equal(t, item, key, msgAndArgs...)
				}

				// Queue should now be empty.
				assert.Equal(t, 0, queue.Len(), msgAndArgs...)
			}
		})
	}
}

type enqueueHandler struct {
	queue []string
	mu    sync.RWMutex
}

func newEnqueueHandler() *enqueueHandler {
	return &enqueueHandler{}
}

var _ croncontroller.EnqueueHandler = (*enqueueHandler)(nil)

func (h *enqueueHandler) EnqueueJobConfig(jobConfig *execution.JobConfig, scheduleTime time.Time) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.queue = append(h.queue, keyFunc(jobConfig, scheduleTime))
	return nil
}

func (h *enqueueHandler) Get() (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.queue) == 0 {
		return "", false
	}
	item := h.queue[0]
	h.queue = h.queue[1:]
	return item, true
}

func (h *enqueueHandler) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.queue)
}

func keyFunc(jobConfig *execution.JobConfig, scheduleTime time.Time) string {
	key, err := croncontroller.JobConfigKeyFunc(jobConfig, scheduleTime)
	if err != nil {
		panic(err)
	}
	return key
}

type notifyingUpdateHandler struct {
	handler croncontroller.UpdateHandler
	cond    chan struct{}
}

func newNotifyingUpdateHandler(handler croncontroller.UpdateHandler) *notifyingUpdateHandler {
	return &notifyingUpdateHandler{
		handler: handler,
		cond:    make(chan struct{}),
	}
}

var _ croncontroller.UpdateHandler = (*notifyingUpdateHandler)(nil)

func (h *notifyingUpdateHandler) Wait() {
	<-h.cond
}

func (h *notifyingUpdateHandler) OnUpdate(jobConfig *execution.JobConfig) {
	h.handler.OnUpdate(jobConfig)
	h.cond <- struct{}{}
}
