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

package schedule

import (
	"fmt"
	"time"

	"github.com/furiko-io/cronexpr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
	"github.com/furiko-io/furiko/pkg/core/tzutils"
	"github.com/furiko-io/furiko/pkg/execution/util/cronparser"
	"github.com/furiko-io/furiko/pkg/utils/heap"
)

// Schedule stores a list of JobConfigs that should be scheduled, and returns
// any JobConfigs that are not yet scheduled.
//
// This data structure assumes that time never goes backwards (i.e. is
// monotonically increasing) between successive calls. In case of clock skew
// which moves the clock backwards, any already scheduled jobs will be simply
// skipped. This data structure is not thread-safe, nor does it provide any sort
// of concurrency control.
type Schedule struct {
	jobConfigs *heap.Heap
	cfg        Config
	clock      clock.Clock
}

// New returns a new Schedule initialized from a list of JobConfigs.
func New(jobConfigs []*execution.JobConfig, options ...Option) (*Schedule, error) {
	sched := &Schedule{
		cfg:   &defaultConfig{},
		clock: clock.RealClock{},
	}

	for _, option := range options {
		option(sched)
	}

	cronCfg, err := sched.cfg.Cron()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load cron config")
	}

	now := sched.clock.Now()
	parser := cronparser.NewParser(cronCfg)
	items := make([]*heap.Item, 0, len(jobConfigs))
	for _, jobConfig := range jobConfigs {
		item, err := sched.newItem(jobConfig, cronCfg, parser, now)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot initialize item in heap")
		}
		if item != nil {
			items = append(items, item)
		}
	}

	sched.jobConfigs = heap.New(items)
	return sched, nil
}

// Pop returns the namespaced name and time of the next item to be scheduled
// that is not after popTime. The returned JobConfig, if any, will be removed
// from the heap.
func (s *Schedule) Pop(fromTime time.Time) (string, time.Time, bool) {
	// NOTE(irvinlim): This whole routine is not thread-safe.

	// Peek at the top-most item in the heap.
	item, ok := s.jobConfigs.Peek()
	if !ok {
		return "", time.Time{}, false
	}

	// Next item is in the future.
	// NOTE(irvinlim): We lose the timezone information
	next := time.Unix(int64(item.Priority()), 0)
	if next.After(fromTime) {
		return "", time.Time{}, false
	}

	// Actually pop the item.
	s.jobConfigs.Pop()

	return item.Name(), next, true
}

// Delete will remove the JobConfig from the internal heap. If the JobConfig did
// not exist, this is a no-op.
func (s *Schedule) Delete(jobConfig *execution.JobConfig) error {
	name, err := cache.MetaNamespaceKeyFunc(jobConfig)
	if err != nil {
		return errors.Wrapf(err, "cannot get namespaced name for jobconfig %v", jobConfig)
	}

	s.jobConfigs.Delete(name)
	return nil
}

// Bump will set the JobConfig's next scheduled time in the internal heap,
// relative to but after popTime. If the JobConfig is in the heap previously,
// it will be updated.
//
// A typical pattern would involve:
//
//  1. Call Pop() to get the next JobConfig to be scheduled. This causes it to be
//     removed from the heap.
//  2. Actually schedule the JobConfig.
//  3. Call Bump() to "commit" the update, which will add it back to the internal
//     heap with a new priority.
//
// Ideally, the caller should only call Bump after the JobConfig has been
// removed (via Pop or Delete) from the heap.
func (s *Schedule) Bump(jobConfig *execution.JobConfig, fromTime time.Time) (time.Time, error) {
	var next time.Time

	name, err := cache.MetaNamespaceKeyFunc(jobConfig)
	if err != nil {
		return next, errors.Wrapf(err, "cannot get namespaced name for jobconfig %v", jobConfig)
	}

	cronCfg, err := s.cfg.Cron()
	if err != nil {
		return next, errors.Wrapf(err, "cannot load cron config")
	}

	parser := cronparser.NewParser(cronCfg)
	expr, timezone, err := s.parseCronAndTimezone(jobConfig, parser)
	if err != nil {
		return next, err
	}

	// Set the timezone from job config if specified, otherwise fall back to the
	// given time.Location of the input time.
	if timezone != nil {
		fromTime = fromTime.In(timezone)
	}

	// Get next schedule time.
	if expr != nil {
		next = getNext(jobConfig, expr, fromTime)
	}

	// There is no next schedule time, so we just return here. Delete it from the
	// heap in case it exists, since there is no next schedule time.
	if next.IsZero() {
		return next, s.Delete(jobConfig)
	}

	// Ensure that next time advances.
	if !next.After(fromTime) {
		return next, fmt.Errorf("next schedule time %v is not after popTime %v", next, fromTime)
	}

	// Create or update item in the heap.
	if _, ok := s.jobConfigs.Search(name); ok {
		s.jobConfigs.Update(name, int(next.Unix()))
	} else {
		s.jobConfigs.Push(name, int(next.Unix()))
	}

	return next, nil
}

// newItem is a helper method that converts a JobConfig into a *heap.Item.
func (s *Schedule) newItem(
	jobConfig *execution.JobConfig,
	cfg *configv1alpha1.CronExecutionConfig,
	parser *cronparser.Parser,
	now time.Time,
) (*heap.Item, error) {
	expr, timezone, err := s.parseCronAndTimezone(jobConfig, parser)
	if err != nil {
		return nil, err
	}

	// Not automatically scheduled.
	if expr == nil || timezone == nil {
		return nil, nil
	}

	// Set the timezone correctly.
	now = now.In(timezone)

	// Evaluate the initial time for scheduling.
	fromTime := getInitialTimeForScheduling(jobConfig, cfg, now, now)

	// Evaluate the next schedule time.
	next := getNext(jobConfig, expr, fromTime)

	var item *heap.Item
	if !next.IsZero() {
		name, err := cache.MetaNamespaceKeyFunc(jobConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get namespaced name for jobconfig %v", jobConfig)
		}
		item = heap.NewItem(name, int(next.Unix()))
	}
	return item, nil
}

func (s *Schedule) parseCronAndTimezone(
	jobConfig *execution.JobConfig,
	parser *cronparser.Parser,
) (*cronexpr.Expression, *time.Location, error) {
	schedule := jobConfig.Spec.Schedule
	if schedule == nil || schedule.Disabled || schedule.Cron == nil || schedule.Cron.Expression == "" {
		return nil, nil, nil
	}

	cfg, err := s.cfg.Cron()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot load cron config")
	}

	// Parse cron expression.
	name, err := cache.MetaNamespaceKeyFunc(jobConfig)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot get namespaced name for jobconfig %v", jobConfig)
	}
	expr, err := parser.Parse(schedule.Cron.Expression, name)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot parse cron schedule: %v", schedule.Cron.Expression)
	}

	// Get time in configured timezone.
	tzstring := getTimezone(schedule.Cron, cfg)
	timezone, err := tzutils.ParseTimezone(tzstring)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot parse timezone: %v", tzstring)
	}

	return expr, timezone, nil
}

// getInitialTimeForScheduling returns the initial time to use as popTime as
// the parameter for getNext.
func getInitialTimeForScheduling(
	jobConfig *execution.JobConfig,
	cfg *configv1alpha1.CronExecutionConfig,
	fromTime, now time.Time,
) time.Time {
	timezone := fromTime.Location()

	maxDowntimeThreshold := time.Second * config.DefaultCronMaxDowntimeThresholdSeconds
	if cfg.MaxDowntimeThresholdSeconds > 0 {
		maxDowntimeThreshold = time.Second * time.Duration(cfg.MaxDowntimeThresholdSeconds)
	}

	// Use the last schedule time in the JobConfig's status to back-schedule all
	// missed schedules (up to a threshold).
	lastScheduleTime := jobConfig.Status.LastScheduled
	if lastScheduleTime != nil && !lastScheduleTime.IsZero() {
		if now.Sub(lastScheduleTime.Time) > maxDowntimeThreshold {
			newLastScheduleTime := metav1.NewTime(now.Add(-maxDowntimeThreshold))
			lastScheduleTime = &newLastScheduleTime
		}
		fromTime = lastScheduleTime.Time
	}

	// Bump to LastUpdated to avoid accidental back-scheduling.
	if spec := jobConfig.Spec.Schedule; spec != nil {
		if lastUpdated := spec.LastUpdated; !lastUpdated.IsZero() && lastUpdated.After(fromTime) {
			fromTime = lastUpdated.Time
		}
	}

	// Cannot schedule before NotBefore.
	// NOTE(irvinlim): Compute next using popTime, so we sub nanosecond in case time falls exactly on NotBefore
	if spec := jobConfig.Spec.Schedule; spec != nil && spec.Constraints != nil {
		if nbf := spec.Constraints.NotBefore; !nbf.IsZero() && fromTime.Before(nbf.Time) {
			fromTime = nbf.Time.Add(-time.Nanosecond)
		}
	}

	// Set timezone correctly.
	fromTime = fromTime.In(timezone)

	return fromTime
}

// getNext is a helper function that evaluates the next closest time immediately
// following popTime, such that it complies with the constraints of the
// JobConfig's schedule.
func getNext(jobConfig *execution.JobConfig, expr *cronexpr.Expression, fromTime time.Time) time.Time {
	next := expr.Next(fromTime)

	// Cannot schedule after NotAfter.
	if spec := jobConfig.Spec.Schedule; spec != nil && spec.Constraints != nil {
		if naf := spec.Constraints.NotAfter; !naf.IsZero() && next.After(naf.Time) {
			next = time.Time{}
		}
	}

	return next
}

// getTimezone returns the timezone for the given JobConfig.
func getTimezone(cronSchedule *execution.CronSchedule, cfg *configv1alpha1.CronExecutionConfig) string {
	// Read from spec.
	if cronSchedule.Timezone != "" {
		return cronSchedule.Timezone
	}

	// Use default timezone from config.
	if tz := cfg.DefaultTimezone; tz != nil && len(*tz) > 0 {
		return *tz
	}

	// Fallback to controller default.
	return config.DefaultCronTimezone
}
