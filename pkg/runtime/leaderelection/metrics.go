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

package leaderelection

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	promNamespace = "furiko"
)

var (
	controllerLeader = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "controller_manager_leader",
			Help:      "Whether the current controller manager instance is elected and running as leader",
		},
		[]string{"lease_name", "lease_id"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		controllerLeader,
	)
}

func ObserveLeaderElected(leaseName, leaseID string, isLeader bool) {
	if isLeader {
		controllerLeader.WithLabelValues(leaseName, leaseID).Set(1)
	} else {
		controllerLeader.WithLabelValues(leaseName, leaseID).Set(0)
	}
}
