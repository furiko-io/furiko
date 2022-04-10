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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/croncontroller"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	getTimeout = time.Millisecond * 100
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
)

type CronWorkerStep struct {
	Name   string
	Time   *CronWorkerTimeStep
	Update *CronWorkerUpdateStep
	Delete *CronWorkerUpdateStep
}

type CronWorkerTimeStep struct {
	Now         time.Time
	WantEnqueue []string
}

type CronWorkerUpdateStep struct {
	JobConfig *execution.JobConfig
}

func TestNewCronWorker(t *testing.T) { // nolint:gocognit
	tests := []struct {
		name       string
		jobConfigs []*execution.JobConfig
		log        klog.Level
		steps      []CronWorkerStep
	}{
		{
			name: "Scheduled every minute",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfig,
			},
			steps: []CronWorkerStep{
				{
					Name: "Initial time",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:52:04Z"),
					},
				},
				{
					Name: "No enqueue until end of minute",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:52:59Z"),
					},
				},
				{
					Name: "Want enqueue at start of minute",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:53:00Z"),
						WantEnqueue: []string{
							keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:53:00Z")),
						},
					},
				},
				{
					Name: "No more enqueue at next second",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:53:01Z"),
					},
				},
				{
					Name: "Want enqueue at start of next minute",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:54:00Z"),
						WantEnqueue: []string{
							keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:54:00Z")),
						},
					},
				},
				{
					Name: "Multiple enqueue when jumping a few minutes",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:57:00Z"),
						WantEnqueue: []string{
							keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:55:00Z")),
							keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:56:00Z")),
							keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:57:00Z")),
						},
					},
				},
			},
		},
		{
			name: "Update JobConfig",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfig,
			},
			steps: []CronWorkerStep{
				{
					Name: "Initial time",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:52:04Z"),
					},
				},
				{
					Name: "Enqueue at next minute",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:53:00Z"),
						WantEnqueue: []string{
							keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:53:00Z")),
						},
					},
				},
				{
					Name: "Update time",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:53:10Z"),
					},
				},
				{
					Name: "Update JobConfig",
					Update: &CronWorkerUpdateStep{
						JobConfig: cronWorkerJobConfigUpdateToEvery15Sec,
					},
				},
				{
					Name: "No enqueue yet",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:53:11Z"),
					},
				},
				{
					Name: "Want enqueue at 15 sec mark",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:53:15Z"),
						WantEnqueue: []string{
							keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:53:15Z")),
						},
					},
				},
			},
		},
		{
			name: "Delete JobConfig",
			jobConfigs: []*execution.JobConfig{
				cronWorkerJobConfig,
			},
			steps: []CronWorkerStep{
				{
					Name: "Initial time",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:52:04Z"),
					},
				},
				{
					Name: "Enqueue at next minute",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:53:00Z"),
						WantEnqueue: []string{
							keyFunc(cronWorkerJobConfig, testutils.Mktime("2022-04-01T10:53:00Z")),
						},
					},
				},
				{
					Name: "Delete JobConfig",
					Delete: &CronWorkerUpdateStep{
						JobConfig: cronWorkerJobConfig,
					},
				},
				{
					Name: "No enqueue 1 minute later",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:54:00Z"),
					},
				},
				{
					Name: "No enqueue 2 minutes later",
					Time: &CronWorkerTimeStep{
						Now: testutils.Mktime("2022-04-01T10:55:00Z"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutils.SetLogLevel(tt.log)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var prevtime time.Time
			fakeClock := clock.NewFakeClock(prevtime)
			croncontroller.Clock = fakeClock

			// Initialize contexts and worker
			c := mock.NewContext()
			ctrlContext := croncontroller.NewContext(c)
			worker := croncontroller.NewCronWorker(ctrlContext)
			executionClient := c.MockClientsets().Furiko().ExecutionV1alpha1()
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

			// Start worker
			worker.Start(ctx)
			time.Sleep(realTimeProcessingDelay)

			for _, step := range tt.steps {
				msgAndArgs := []interface{}{
					fmt.Sprintf("Error in step: %v", step.Name),
				}

				// Perform time step.
				if step := step.Time; step != nil {
					// Update clock, ensure to never move backwards in time.
					if !step.Now.IsZero() {
						if prevtime.After(step.Now) {
							t.Errorf("cannot set now to %v, previous time was %v", step.Now, prevtime)
							return
						}
						prevtime = step.Now
						fakeClock.SetTime(step.Now)
						time.Sleep(realTimeProcessingDelay)
					}

					// Consume from queue or timeout.
					for _, key := range step.WantEnqueue {
						item, _, err := testutils.WorkqueueGetWithTimeout(worker.Queue, getTimeout)
						if err != nil {
							assert.NoError(t, err, msgAndArgs...)
							break
						}
						assert.Equal(t, item, key, msgAndArgs...)
					}

					// Queue should now be empty.
					assert.Equal(t, 0, worker.Queue.Len(), msgAndArgs...)
				}

				// Perform update step.
				if step := step.Update; step != nil {
					updatedJobConfig, err := executionClient.JobConfigs(step.JobConfig.Namespace).
						Update(ctx, step.JobConfig, metav1.UpdateOptions{})
					assert.NoError(t, err)

					// Trigger update in worker.
					worker.UpdatedConfigs <- updatedJobConfig
				}

				// Perform delete step.
				if step := step.Delete; step != nil {
					err := executionClient.JobConfigs(step.JobConfig.Namespace).
						Delete(ctx, step.JobConfig.Name, metav1.DeleteOptions{})
					assert.NoError(t, err)

					// Trigger update in worker.
					worker.UpdatedConfigs <- step.JobConfig
				}
			}
		})
	}
}

func keyFunc(jobConfig *execution.JobConfig, scheduleTime time.Time) string {
	key, err := croncontroller.JobConfigKeyFunc(jobConfig, scheduleTime)
	if err != nil {
		panic(err)
	}
	return key
}
