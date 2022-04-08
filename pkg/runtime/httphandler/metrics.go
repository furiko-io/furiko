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

package httphandler

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
)

const (
	defaultMetricsPath = "/metrics"
)

// ServeMetrics adds Prometheus metrics handlers to the given serve mux.
func ServeMetrics(mux *http.ServeMux, cfg *configv1alpha1.MetricsSpec) {
	// Not enabled.
	if cfg == nil {
		return
	}
	if enabled := cfg.Enabled; enabled == nil || !*enabled {
		return
	}

	path := cfg.MetricsPath
	if path == "" {
		path = defaultMetricsPath
	}

	// Use registry from controller-runtime
	mux.Handle(path, promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.HTTPErrorOnError,
	}))

	klog.V(4).Infof("httphandler: added http handler for metrics")
}
