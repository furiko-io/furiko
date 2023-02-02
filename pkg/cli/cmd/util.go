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

package cmd

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/execution/util/cron"
	"github.com/furiko-io/furiko/pkg/execution/util/schedule"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

// FormatNextSchedule formats a JobConfig's next schedule.
func FormatNextSchedule(
	jobConfig *execution.JobConfig,
	cronParser *cron.Parser,
	format formatter.TimeFormatFunc,
) (string, error) {
	nextSchedule := "Never"
	if format == nil {
		format = formatter.FormatTime
	}

	hashID, err := cache.MetaNamespaceKeyFunc(jobConfig)
	if err != nil {
		return "", errors.Wrapf(err, "cannot get namespaced name")
	}

	// Check if there is no schedule.
	scheduleSpec := jobConfig.Spec.Schedule
	if scheduleSpec == nil || scheduleSpec.Cron == nil || len(scheduleSpec.Cron.GetExpressions()) == 0 {
		return nextSchedule, nil
	}

	// Prepare cron expression.
	expr, err := cron.NewExpressionFromCronSchedule(scheduleSpec.Cron, cronParser, hashID)
	if err != nil {
		return "", errors.Wrapf(err, "cannot parse cron schedule")
	}

	// Get next schedule time.
	next := expr.Next(ktime.Now().Time)
	if next.IsZero() {
		return nextSchedule, nil
	}

	// Not allowed to be scheduled by policy.
	if !schedule.IsAllowed(scheduleSpec, next) {
		return nextSchedule, nil
	}

	// Otherwise, format and return the next schedule time.
	t := metav1.NewTime(next)
	nextSchedule = format(&t)
	return nextSchedule, nil
}
