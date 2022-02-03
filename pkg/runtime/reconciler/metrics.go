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

package reconciler

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	promNamespace = "furiko"
)

var (
	controllerWorkDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: promNamespace,
			Name:      "controller_work_duration_seconds",
			Help:      "Time taken for a single controller worker run",
			// Buckets are taken from controller-runtime.
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.45, 0.5, 0.6, 0.7, 0.8,
				0.9, 1.0, 1.25, 1.5, 1.75, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5, 6, 7, 8, 9, 10, 15, 20, 25, 30, 40, 50, 60},
		},
		[]string{"controller_name"},
	)

	controllerSyncErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "controller_sync_errors_total",
			Help:      "Total number of errors from controller sync workers",
		},
		[]string{"controller_name", "error_kind"},
	)

	controllerRetriesExceededTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "controller_retries_exceeded_total",
			Help:      "Total number of items which exceeded retries on reconcile and then gave up",
		},
		[]string{"controller_name"},
	)

	controllerWorkersTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "controller_workers_total",
			Help:      "Total number of controller workers spawned",
		},
		[]string{"controller_name"},
	)

	controllerWorkersActiveTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "controller_workers_active_total",
			Help:      "Total number of controller workers that are currently actively working",
		},
		[]string{"controller_name"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		controllerWorkDuration,
		controllerSyncErrorsTotal,
		controllerRetriesExceededTotal,
		controllerWorkersTotal,
		controllerWorkersActiveTotal,
	)
}

func ObserveWorkersTotal(name string, concurrency int) {
	controllerWorkersTotal.WithLabelValues(name).Set(float64(concurrency))
}

func ObserveWorkerStart(name string) {
	controllerWorkersActiveTotal.WithLabelValues(name).Inc()
}

func ObserveWorkerEnd(name string, duration time.Duration) {
	controllerWorkersActiveTotal.WithLabelValues(name).Dec()
	controllerWorkDuration.WithLabelValues(name).Observe(duration.Seconds())
}

func ObserveControllerSyncError(name string, kind string) {
	controllerSyncErrorsTotal.WithLabelValues(name, kind).Inc()
}

func ObserveRetriesExceeded(name string) {
	controllerRetriesExceededTotal.WithLabelValues(name).Inc()
}
