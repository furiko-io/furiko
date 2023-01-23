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

package croncontroller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
	utiltrace "k8s.io/utils/trace"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
	"github.com/furiko-io/furiko/pkg/execution/util/schedule"
)

const (
	// cronWorkerInterval is the interval between checking if JobConfigs should be
	// enqueued. We only really need this as small as once per second.
	cronWorkerInterval = time.Second
)

var (
	Clock clock.Clock = &clock.RealClock{}
)

// CronWorker enqueues Job names to be scheduled, based on the cron schedule of the config.
// It will enqueue one item for each schedule interval, which is a 1:1 correspondence with a Job
// to be created.
type CronWorker struct {
	*Context
	schedule *schedule.Schedule
	handler  EnqueueHandler
	mu       sync.Mutex
}

// EnqueueHandler knows how to enqueue a JobConfig to be created.
type EnqueueHandler interface {
	EnqueueJobConfig(jobConfig *execution.JobConfig, scheduleTime time.Time) error
}

func NewCronWorker(ctrlContext *Context, handler EnqueueHandler) *CronWorker {
	return &CronWorker{
		Context: ctrlContext,
		handler: handler,
	}
}

func (w *CronWorker) WorkerName() string {
	return fmt.Sprintf("%v.CronWorker", controllerName)
}

// Init is called before we start the worker. We will initialize all JobConfigs
// into the Schedule.
func (w *CronWorker) Init() error {
	jobConfigs, err := w.jobconfigInformer.Lister().JobConfigs(metav1.NamespaceAll).List(labels.Everything())
	if err != nil {
		return errors.Wrapf(err, "cannot list all jobconfigs")
	}

	sched, err := schedule.New(
		jobConfigs,
		schedule.WithClock(Clock),
		schedule.WithConfigLoader(w.Configs()),
	)
	if err != nil {
		return err
	}

	w.schedule = sched
	return nil
}

func (w *CronWorker) Start(ctx context.Context) {
	go ClockTickUntil(instrumentWorkerMetrics(w.WorkerName(), w.Work), cronWorkerInterval, ctx.Done())
}

// Work runs a single iteration of synchronizing all JobConfigs.
func (w *CronWorker) Work() {
	w.mu.Lock()
	defer w.mu.Unlock()

	trace := utiltrace.New(
		"cron_schedule_all",
	)

	// NOTE(irvinlim): Log if it takes longer than the worker interval to run the
	// entire work routine.
	defer trace.LogIfLong(cronWorkerInterval)

	// Load dynamic configuration.
	cfg, err := w.Configs().Cron()
	if err != nil {
		klog.ErrorS(err, "croncontroller: cannot load controller configuration")
		return
	}

	now := Clock.Now()
	maxMissedSchedules := config.DefaultCronMaxMissedSchedules
	if spec := cfg.MaxMissedSchedules; spec != nil {
		maxMissedSchedules = int(*spec)
	}
	scheduledCount := make(map[string]int)

	// Flush all JobConfigs that had been updated.
	w.refreshUpdatedJobConfigs(now)
	trace.Step("Flushing of JobConfig updates done")

	var scheduled uint64
	for {
		// Get the next job config that is due for scheduling, otherwise return early.
		key, ts, ok := w.schedule.Pop(Clock.Now())
		if !ok {
			break
		}

		// Should not happen.
		if ts.IsZero() {
			continue
		}

		// Enqueue the job.
		if err := w.syncOne(now, key, ts, scheduledCount, maxMissedSchedules); err != nil {
			klog.ErrorS(err, "croncontroller: cannot sync single jobconfig to be scheduled",
				"worker", w.WorkerName(),
				"key", key,
				"scheduleTime", ts,
			)
			continue
		}

		scheduled++
		trace.Step("Sync JobConfig to be scheduled")
	}

	if scheduled > 0 {
		klog.V(4).InfoS("croncontroller: successfully scheduled jobs", "count", scheduled)
	}
}

// refreshUpdatedJobConfigs will read all keys to be flushed, and recompute the
// next schedule time for all job configs that were updated.
func (w *CronWorker) refreshUpdatedJobConfigs(now time.Time) {
	flushes := 0
	defer func() {
		if flushes > 0 {
			klog.V(4).InfoS("croncontroller: refreshed all job configs that need updating",
				"worker", w.WorkerName(),
				"len", flushes,
			)
		}
	}()

	// Perform at most 1000 flushes per iteration to prevent backlogging.
	for flushes < 1000 {
		select {
		case jobConfig := <-w.updatedConfigs:
			flushes++

			// Delete and add it back to the heap. We use the current time as the reference
			// time, assuming that its previous schedule time is in the future.
			if err := w.schedule.Delete(jobConfig); err != nil {
				klog.ErrorS(err, "croncontroller: cannot remove outdated job config from heap",
					"namespace", jobConfig.Namespace,
					"name", jobConfig.Name,
				)
				continue
			}
			if _, err := w.schedule.Bump(jobConfig, now); err != nil {
				klog.ErrorS(err, "croncontroller: cannot bump updated job config in heap",
					"namespace", jobConfig.Namespace,
					"name", jobConfig.Name,
				)
				continue
			}
		default:
			// Nothing more to flush.
			return
		}
	}
}

// syncOne reconciles a single JobConfig and enqueues Jobs to be created.
func (w *CronWorker) syncOne(now time.Time, key string, ts time.Time, counts map[string]int, maxCount int) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return errors.Wrapf(err, "cannot split namespace and name from key %v", key)
	}
	jobConfig, err := w.jobconfigInformer.Lister().JobConfigs(namespace).Get(name)
	if err != nil {
		return errors.Wrapf(err, "cannot get jobconfig %v", key)
	}

	// Check that it does not exceed maximum backschedule limit.
	if counts[key] >= maxCount {
		// Bump to next schedule time.
		newNext, err := w.schedule.Bump(jobConfig, now)
		if err != nil {
			return errors.Wrapf(err, "cannot bump to new next schedule time")
		}

		klog.ErrorS(
			errors.New("missed too many schedules"),
			"croncontroller: skipping back-scheduling for JobConfig",
			"worker", w.WorkerName(),
			"namespace", jobConfig.GetNamespace(),
			"name", jobConfig.GetName(),
			"maxCount", maxCount,
			"newNextSchedule", newNext,
		)

		return nil
	}

	// Update counter.
	counts[key]++

	// Enqueue the job.
	if err := w.handler.EnqueueJobConfig(jobConfig, ts); err != nil {
		return errors.Wrapf(err, "cannot enqueue job for %v", ts)
	}

	klog.V(3).InfoS("croncontroller: scheduled job",
		"worker", w.WorkerName(),
		"namespace", jobConfig.GetNamespace(),
		"name", jobConfig.GetName(),
		"scheduleTime", ts,
	)

	// Bump to next schedule time.
	newNext, err := w.schedule.Bump(jobConfig, ts)
	if err != nil {
		return errors.Wrapf(err, "cannot bump to new next schedule time")
	}

	klog.V(6).InfoS("croncontroller: bumped job config's next schedule time",
		"worker", w.WorkerName(),
		"namespace", jobConfig.GetNamespace(),
		"name", jobConfig.GetName(),
		"previous", ts,
		"next", newNext,
	)

	return nil
}

// getTimezone returns the timezone for the given JobConfig.
func (w *CronWorker) getTimezone(cronSchedule *execution.CronSchedule, cfg *configv1alpha1.CronExecutionConfig) string {
	// Read from spec.
	if cronSchedule.Timezone != "" {
		return cronSchedule.Timezone
	}

	// Use default timezone from config.
	if tz := cfg.DefaultTimezone; tz != nil && len(*tz) > 0 {
		return *tz
	}

	// Fallback to controller default.
	return defaultTimezone
}

type enqueueHandler struct {
	*Context
}

var _ EnqueueHandler = (*enqueueHandler)(nil)

func newEnqueueHandler(ctrlContext *Context) *enqueueHandler {
	return &enqueueHandler{
		Context: ctrlContext,
	}
}

func (h *enqueueHandler) EnqueueJobConfig(jobConfig *execution.JobConfig, scheduleTime time.Time) error {
	// Use our custom KeyFunc.
	key, err := JobConfigKeyFunc(jobConfig, scheduleTime)
	if err != nil {
		return errors.Wrapf(err, "keyfunc error")
	}

	// Add the (JobConfig, ScheduleTime) to the workqueue.
	h.queue.Add(key)

	return nil
}
