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

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/httphandler"
	httphandlertesting "github.com/furiko-io/furiko/pkg/runtime/httphandler/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

var (
	defaultHealthHandler = &mockHealthHandler{
		healthStatus: []controllermanager.HealthStatus{
			{
				Name:    "TestController",
				Healthy: true,
			},
		},
	}
)

type mockHealthHandler struct {
	readiness    error
	healthStatus []controllermanager.HealthStatus
}

var _ httphandler.HealthHandler = (*mockHealthHandler)(nil)

func (m *mockHealthHandler) GetReadiness() error {
	return m.readiness
}

func (m *mockHealthHandler) GetHealth() []controllermanager.HealthStatus {
	return m.healthStatus
}

func TestServeHealth(t *testing.T) {
	makeTestCasesFor404 := func() []*httphandlertesting.TestCase {
		return []*httphandlertesting.TestCase{
			{
				Name:     "readiness probe path should not be enabled",
				Path:     "/readyz",
				WantCode: 404,
			},
			{
				Name:     "liveness probe path should not be enabled",
				Path:     "/healthz",
				WantCode: 404,
			},
		}
	}

	makeTestCasesFor200 := func(readinessPath, livenessPath string) []*httphandlertesting.TestCase {
		return []*httphandlertesting.TestCase{
			{
				Name:     "readiness probe should be ok",
				Path:     readinessPath,
				WantCode: 200,
				AssertBody: testutils.AssertValueAll(
					testutils.AssertValueContains("ok"),
				),
			},
			{
				Name:     "liveness probe should be ok",
				Path:     livenessPath,
				WantCode: 202,
				AssertBody: testutils.AssertValueAll(
					testutils.AssertValueJSONEq(map[string]any{
						"healthy": true,
						"status":  defaultHealthHandler.healthStatus,
					}),
				),
			},
		}
	}

	type testCase struct {
		name    string
		cfg     *configv1alpha1.HTTPHealthSpec
		handler httphandler.HealthHandler
		cases   []*httphandlertesting.TestCase
	}
	for _, tt := range []testCase{
		{
			name:    "default config",
			handler: defaultHealthHandler,
			cases:   makeTestCasesFor404(),
		},
		{
			name: "explicitly disabled",
			cfg: &configv1alpha1.HTTPHealthSpec{
				Enabled: pointer.Bool(false),
			},
			handler: defaultHealthHandler,
			cases:   makeTestCasesFor404(),
		},
		{
			name:    "handler is enabled",
			handler: defaultHealthHandler,
			cfg: &configv1alpha1.HTTPHealthSpec{
				Enabled: pointer.Bool(true),
			},
			cases: makeTestCasesFor200("/readyz", "/healthz"),
		},
		{
			name:    "custom probe paths",
			handler: defaultHealthHandler,
			cfg: &configv1alpha1.HTTPHealthSpec{
				Enabled:            pointer.Bool(true),
				LivenessProbePath:  "/liveness",
				ReadinessProbePath: "/readiness",
			},
			cases: append(
				makeTestCasesFor404(),
				makeTestCasesFor200("/readiness", "/liveness")...,
			),
		},
		{
			name: "error scenarios",
			handler: &mockHealthHandler{
				readiness: assert.AnError,
				healthStatus: []controllermanager.HealthStatus{
					{
						Name:    "TestController",
						Healthy: false,
						Message: "Controller is deadlocked",
					},
				},
			},
			cfg: &configv1alpha1.HTTPHealthSpec{
				Enabled: pointer.Bool(true),
			},
			cases: []*httphandlertesting.TestCase{
				{
					Name:     "readiness probe returns error",
					Path:     "/readyz",
					WantCode: 503,
				},
				{
					Name:     "liveness probe returns unhealthy status",
					Path:     "/healthz",
					WantCode: 500,
					AssertBody: testutils.AssertValueAll(
						testutils.AssertValueJSONEq(map[string]any{
							"healthy": false,
							"status": []controllermanager.HealthStatus{
								{
									Name:    "TestController",
									Healthy: false,
									Message: "Controller is deadlocked",
								},
							},
						}),
					),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			suite := httphandlertesting.NewSuite(&httphandlertesting.TestSuite{
				Method: http.MethodGet,
			})
			httphandler.ServeHealth(suite.GetMux(), tt.cfg, tt.handler)
			suite.Run(t, tt.cases)
		})
	}
}
