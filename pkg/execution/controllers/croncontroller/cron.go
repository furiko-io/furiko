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
	"fmt"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	utiltrace "k8s.io/utils/trace"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/tzutils"
	"github.com/furiko-io/furiko/pkg/execution/util/cronparser"
)

const (
	// CronWorkerInterval is the interval between checking if JobConfigs should be
	// enqueued. We only really need this as small as once per second.
	CronWorkerInterval = time.Second
)

// CronWorker enqueues Job names to be scheduled, based on the cron schedule of the config.
// It will enqueue one item for each schedule interval, which is a 1:1 correspondence with a Job
// to be created.
type CronWorker struct {
	*Context
	*Schedule
}

func NewCronWorker(ctrlContext *Context) *CronWorker {
	return &CronWorker{
		Context:  ctrlContext,
		Schedule: NewSchedule(ctrlContext),
	}
}

func (w *CronWorker) WorkerName() string {
	return fmt.Sprintf("%v.CronWorker", controllerName)
}

func (w *CronWorker) Start(stopCh <-chan struct{}) {
	go CronTimerUntil(instrumentWorkerMetrics(w.WorkerName(), w.work), CronWorkerInterval, stopCh)
}

// work runs a single iteration of synchronizing JobConfigs.
func (w *CronWorker) work() {
	trace := utiltrace.New(
		"cron_schedule_all",
	)
	defer trace.LogIfLong(CronWorkerInterval / 2)

	// Load dynamic configuration.
	cfg, err := w.Configs().CronController()
	if err != nil {
		klog.ErrorS(err, "croncontroller: cannot load controller configuration")
		return
	}

	// Get all keys to be flushed.
	w.flushKeys()
	trace.Step("Flushing of JobConfig updates done")

	// Get all JobConfigs from the cache.
	jobConfigList, err := w.jobconfigInformer.Lister().JobConfigs(metav1.NamespaceAll).List(labels.Everything())
	if err != nil {
		klog.ErrorS(err, "croncontroller: list JobConfig error", "worker", w.WorkerName())
		return
	}
	trace.Step("List all JobConfigs done")

	// Create cron parser instance
	parser := cronparser.NewParser(cfg)

	// Sync each job config.
	// TODO(irvinlim): Theoretically it is more computationally efficient to use a
	//  heap instead of iterating all job configs for scheduling.
	for _, jobConfig := range jobConfigList {
		if err := w.syncOne(jobConfig, cfg, parser); err != nil {
			klog.ErrorS(err, "croncontroller: sync JobConfig error",
				"worker", w.WorkerName(),
				"namespace", jobConfig.GetNamespace(),
				"name", jobConfig.GetName(),
			)
		}
	}
	trace.Step("Sync all JobConfigs done")
}

// flushKeys will read all keys to be flushed, and flush it from the nextScheduleTime precomputed map.
func (w *CronWorker) flushKeys() {
	flushes := 0
	defer func() {
		if flushes > 0 {
			klog.V(4).InfoS("croncontroller: flushed all job configs that need updating",
				"worker", w.WorkerName(),
				"len", flushes,
			)
		}
	}()

	// Perform at most 1000 flushes per iteration to prevent backlogging.
	for flushes < 1000 {
		select {
		case jobConfig := <-w.updatedConfigs:
			w.FlushNextScheduleTime(jobConfig)
			flushes++
		default:
			// Nothing more to flush.
			return
		}
	}
}

// syncOne reconciles a single JobConfig and enqueues Jobs to be created.
func (w *CronWorker) syncOne(
	jobConfig *execution.JobConfig,
	cfg *configv1.CronControllerConfig,
	parser *cronparser.Parser,
) error {
	schedule := jobConfig.Spec.Schedule
	if schedule == nil || schedule.Disabled || schedule.Cron == nil || len(schedule.Cron.Expression) == 0 {
		return nil
	}

	namespacedName, err := cache.MetaNamespaceKeyFunc(jobConfig)
	if err != nil {
		return errors.Wrapf(err, "cannot get namespaced name")
	}

	expr, err := parser.Parse(schedule.Cron.Expression, namespacedName)
	if err != nil {
		return errors.Wrapf(err, "cannot parse cron schedule: %v", schedule.Cron.Expression)
	}

	// Get time in configured timezone.
	tzstring := w.getTimezone(schedule.Cron, cfg)
	timezone, err := tzutils.ParseTimezone(tzstring)
	if err != nil {
		return errors.Wrapf(err, "cannot parse timezone: %v", tzstring)
	}
	now := Clock.Now().In(timezone)

	maxMissedSchedules := 5
	if cfg.MaxMissedSchedules > 0 {
		maxMissedSchedules = int(cfg.MaxMissedSchedules)
	}

	// Get next scheduled time repeatedly and enqueue for each scheduled time.
	// This helps to prevent missed executions (maybe due to worker stuck).
	for i := 0; i < maxMissedSchedules; i++ {
		next := w.GetNextScheduleTime(jobConfig, now, expr)

		// There is no next schedule time.
		if next.IsZero() {
			return nil
		}

		// Skip if the next schedule is in the future.
		if next.After(now) {
			return nil
		}

		// Enqueue the job.
		if err := w.enqueueJob(jobConfig, next); err != nil {
			return errors.Wrapf(err, "cannot enqueue job for %v", next)
		}

		// Bump the next schedule time for the job config.
		w.BumpNextScheduleTime(jobConfig, next, expr)

		klog.V(2).InfoS("croncontroller: scheduled job by cron",
			"worker", w.WorkerName(),
			"namespace", jobConfig.GetNamespace(),
			"name", jobConfig.GetName(),
			"cron_schedule", schedule.Cron.Expression,
			"schedule_time", next,
			"timezone", timezone,
		)
	}

	// Reached here, means we missed more too many schedules.
	// Just bump next scheduled time.
	err = fmt.Errorf("missed too many schedules, maximum=%v", maxMissedSchedules)
	klog.ErrorS(err, "croncontroller: skipping back-scheduling for JobConfig",
		"worker", w.WorkerName(),
		"namespace", jobConfig.GetNamespace(),
		"name", jobConfig.GetName(),
	)
	w.BumpNextScheduleTime(jobConfig, now, expr)

	return nil
}

// enqueueJob enqueues a single job to be scheduled.
func (w *CronWorker) enqueueJob(jobConfig *execution.JobConfig, scheduleTime time.Time) error {
	// Use our custom KeyFunc.
	key, err := JobConfigKeyFunc(jobConfig, scheduleTime)
	if err != nil {
		return errors.Wrapf(err, "keyfunc error")
	}

	// Add the (JobConfig, ScheduleTime) to the workqueue.
	w.queue.Add(key)

	return nil
}

// getTimezone returns the timezone for the given JobConfig.
func (w *CronWorker) getTimezone(cronSchedule *execution.CronSchedule, cfg *configv1.CronControllerConfig) string {
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
