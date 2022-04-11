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
	"sync"
	"time"

	"github.com/furiko-io/cronexpr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

const (
	defaultMaxDowntimeThreshold = time.Minute * 5
)

var (
	// Clock is exposed for unit tests. Note that multiple components may share
	// references to the same clock in this package.
	Clock clock.Clock = clock.RealClock{}
)

// Schedule is a thread-safe store for the next schedule time of JobConfigs.
type Schedule struct {
	ctrlContext controllercontext.Context

	// Internal computed state of next schedule time for each JobConfig.
	nextScheduleTime sync.Map
}

func NewSchedule(ctrlContext controllercontext.Context) *Schedule {
	return &Schedule{
		ctrlContext: ctrlContext,
	}
}

// GetNextScheduleTime returns the next expected schedule time for a JobConfig.
// This encapsulates the core logic for deciding if a Job should be enqueued.
// Returns a zero-value time if the cron expression does not have any timestamp in the future.
// The cron expression is interpreted relative to the timezone of fromTime.
func (w *Schedule) GetNextScheduleTime(
	jobConfig *execution.JobConfig, fromTime time.Time, expr *cronexpr.Expression,
) time.Time {
	timezone := fromTime.Location()

	maxDowntimeThreshold := defaultMaxDowntimeThreshold
	if cfg, err := w.ctrlContext.Configs().Cron(); err == nil && cfg.MaxDowntimeThresholdSeconds > 0 {
		maxDowntimeThreshold = time.Second * time.Duration(cfg.MaxDowntimeThresholdSeconds)
	}

	// Return previously computed value.
	if next := w.getNextScheduleTime(jobConfig); next != nil {
		return *next
	}

	// Did not previously compute next schedule time (or was cleared). Use the last
	// schedule time in the JobConfig's status to back-schedule all missed schedules
	// (up to a threshold).
	lastScheduleTime := jobConfig.Status.LastScheduleTime
	if lastScheduleTime != nil && !lastScheduleTime.IsZero() {
		if Clock.Since(lastScheduleTime.Time) > maxDowntimeThreshold {
			newLastScheduleTime := metav1.NewTime(Clock.Now().Add(-maxDowntimeThreshold))
			lastScheduleTime = &newLastScheduleTime
		}
		fromTime = lastScheduleTime.Time
	}

	// Cannot schedule before schedule.NotBefore.
	if spec := jobConfig.Spec.Schedule; spec != nil && spec.Constraints != nil {
		if nbf := spec.Constraints.NotBefore; !nbf.IsZero() && nbf.After(fromTime) {
			fromTime = nbf.Time
		}
	}

	// Set timezone correctly.
	fromTime = fromTime.In(timezone)

	// Compute the next scheduled time.
	next := expr.Next(fromTime)

	// We update our internal map so it will be fetched when the time comes.
	w.setNextScheduleTime(jobConfig, next)
	return next
}

// BumpNextScheduleTime updates the next schedule time of a JobConfig.
func (w *Schedule) BumpNextScheduleTime(
	jobConfig *execution.JobConfig, fromTime time.Time, expr *cronexpr.Expression,
) {
	next := expr.Next(fromTime)
	w.setNextScheduleTime(jobConfig, next)
}

// FlushNextScheduleTime flushes the next schedule time for a JobConfig, causing it to be recomputed on
// the next time GetNextScheduleTime is called.
func (w *Schedule) FlushNextScheduleTime(jobConfig *execution.JobConfig) {
	w.nextScheduleTime.Delete(getNextScheduleKey(jobConfig))
}

func (w *Schedule) getNextScheduleTime(jobConfig *execution.JobConfig) *time.Time {
	val, ok := w.nextScheduleTime.Load(getNextScheduleKey(jobConfig))
	if !ok {
		return nil
	}
	ts, ok := val.(time.Time)
	if !ok {
		return nil
	}
	return &ts
}

func (w *Schedule) setNextScheduleTime(jobConfig *execution.JobConfig, ts time.Time) {
	if !ts.IsZero() {
		// Prevent setting smaller timestamps as a defensive measure. This could happen
		// due to several reasons: NTP shift backwards after clock skew, a bug in cron
		// expression library producing an older time, etc.
		// TODO(irvinlim): Potential TOCTTOU here
		if prev := w.getNextScheduleTime(jobConfig); prev != nil && ts.Before(*prev) {
			err := fmt.Errorf("next schedule time was decreased from %v to %v", prev, ts)
			klog.ErrorS(err, "croncontroller: prevented next schedule time from going backwards",
				"namespace", jobConfig.GetNamespace(),
				"name", jobConfig.GetName(),
			)
			return
		}
	}

	klog.V(5).InfoS("croncontroller: store next schedule time",
		"namespace", jobConfig.GetNamespace(),
		"name", jobConfig.GetName(),
		"next", ts,
	)

	w.nextScheduleTime.Store(getNextScheduleKey(jobConfig), ts)
}

func getNextScheduleKey(jobConfig *execution.JobConfig) string {
	key := jobConfig.GetName()
	if val, err := cache.MetaNamespaceKeyFunc(jobConfig); err == nil {
		key = val
	}
	return key
}
