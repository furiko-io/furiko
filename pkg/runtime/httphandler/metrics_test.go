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

package httphandler_test

import (
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/httphandler"
	httphandlertesting "github.com/furiko-io/furiko/pkg/runtime/httphandler/testing"
)

func TestServeMetrics(t *testing.T) {
	type testCase struct {
		name  string
		cfg   *configv1alpha1.HTTPMetricsSpec
		cases []*httphandlertesting.TestCase
	}
	for _, tt := range []testCase{
		{
			name: "default config",
			cases: []*httphandlertesting.TestCase{
				{
					Name:     "should not be enabled",
					WantCode: 404,
				},
			},
		},
		{
			name: "explicitly disabled",
			cfg: &configv1alpha1.HTTPMetricsSpec{
				Enabled: pointer.Bool(false),
			},
			cases: []*httphandlertesting.TestCase{
				{
					Name:     "should not be enabled",
					WantCode: 404,
				},
			},
		},
		{
			name: "config is enabled",
			cfg: &configv1alpha1.HTTPMetricsSpec{
				Enabled: pointer.Bool(true),
			},
			cases: []*httphandlertesting.TestCase{
				{
					Name:     "should be ok",
					WantCode: 200,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			suite := httphandlertesting.NewSuite(&httphandlertesting.TestSuite{
				Method: http.MethodGet,
				Path:   "/metrics",
			})
			httphandler.ServeMetrics(suite.GetMux(), tt.cfg, prometheus.NewRegistry())
			suite.Run(t, tt.cases)
		})
	}
}
