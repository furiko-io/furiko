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

package jobcontroller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
)

const (
	// TODO(irvinlim): Support configuration of global metrics namespace.
	promNamespace = "furiko"
)

var (
	firstTaskCreationLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: promNamespace,
			Name:      "first_task_creation_latency",
			Help:      "Latency between the Job's creation time and when its first Task was created",
			Buckets:   prometheus.ExponentialBuckets(0.05, 2, 15),
		},
		[]string{"namespace", "job_type"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		firstTaskCreationLatency,
	)
}

// ObserveFirstTaskCreation records a metric for the first task creation time of a Job.
func ObserveFirstTaskCreation(rj *execution.Job, task tasks.Task) {
	// There are tasks created prior, do not observe.
	if len(rj.Status.Tasks) > 0 {
		return
	}

	expectedStartTime := rj.Status.StartTime

	// TODO(irvinlim): For scheduled jobs, we should measure latency from its scheduled time.

	// Record metric for latency between the expected start time and task creation time.
	latency := task.GetTaskRef().CreationTimestamp.Sub(expectedStartTime.Time)
	firstTaskCreationLatency.WithLabelValues(rj.GetNamespace(), string(rj.Spec.Type)).Observe(latency.Seconds())
}
