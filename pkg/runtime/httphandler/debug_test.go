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
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/httphandler"
	httphandlertesting "github.com/furiko-io/furiko/pkg/runtime/httphandler/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

var (
	mockBootstrapConfig = &configv1alpha1.ExecutionControllerConfig{
		BootstrapConfigSpec: configv1alpha1.BootstrapConfigSpec{
			DefaultResync: metav1.Duration{Duration: time.Hour},
			HTTP: &configv1alpha1.HTTPSpec{
				BindAddress: ":18080",
			},
		},
	}

	mockDynamicConfig = map[configv1alpha1.ConfigName]runtime.Object{
		configv1alpha1.JobExecutionConfigName: &configv1alpha1.JobExecutionConfig{
			DefaultPendingTimeoutSeconds: pointer.Int64(1800),
		},
	}

	defaultDebugHandler = &mockDebugHandler{
		bootstrapConfig: mockBootstrapConfig,
		dynamicConfig:   mockDynamicConfig,
	}
)

type mockDebugHandler struct {
	bootstrapConfig    runtime.Object
	dynamicConfig      map[configv1alpha1.ConfigName]runtime.Object
	dynamicConfigError error
}

var _ httphandler.DebugHandler = (*mockDebugHandler)(nil)

func (m *mockDebugHandler) GetBootstrapConfig() runtime.Object {
	return m.bootstrapConfig
}

func (m *mockDebugHandler) GetDynamicConfig() (map[configv1alpha1.ConfigName]runtime.Object, error) {
	if m.dynamicConfigError != nil {
		return nil, m.dynamicConfigError
	}
	return m.dynamicConfig, nil
}

func TestServeDebug(t *testing.T) {
	makeTestCasesFor404 := func() []*httphandlertesting.TestCase {
		return []*httphandlertesting.TestCase{
			{
				Name:     "bootstrap config should not be enabled",
				Path:     "/debug/config/bootstrap",
				WantCode: 404,
			},
			{
				Name:     "dynamic config should not be enabled",
				Path:     "/debug/config/dynamic",
				WantCode: 404,
			},
		}
	}

	makeTestCasesFor200 := func(basePath string) []*httphandlertesting.TestCase {
		return []*httphandlertesting.TestCase{
			{
				Name: "bootstrap config should not be enabled",
				Path: basePath + "/config/bootstrap",
				AssertBody: testutils.AssertValueAll(
					testutils.AssertValueJSONEq(mockBootstrapConfig),
				),
			},
			{
				Name: "dynamic config should not be enabled",
				Path: basePath + "/config/dynamic",
				AssertBody: testutils.AssertValueAll(
					testutils.AssertValueJSONEq(mockDynamicConfig),
				),
			},
		}
	}

	type testCase struct {
		name    string
		cfg     *configv1alpha1.HTTPDebugSpec
		handler httphandler.DebugHandler
		cases   []*httphandlertesting.TestCase
	}
	for _, tt := range []testCase{
		{
			name:    "default config",
			handler: defaultDebugHandler,
			cases:   makeTestCasesFor404(),
		},
		{
			name: "explicitly disabled",
			cfg: &configv1alpha1.HTTPDebugSpec{
				Enabled: pointer.Bool(false),
			},
			handler: defaultDebugHandler,
			cases:   makeTestCasesFor404(),
		},
		{
			name:    "handler is enabled",
			handler: defaultDebugHandler,
			cfg: &configv1alpha1.HTTPDebugSpec{
				Enabled: pointer.Bool(true),
			},
			cases: makeTestCasesFor200("/debug"),
		},
		{
			name:    "custom base path",
			handler: defaultDebugHandler,
			cfg: &configv1alpha1.HTTPDebugSpec{
				Enabled:  pointer.Bool(true),
				BasePath: "/state",
			},
			cases: append(makeTestCasesFor404(), makeTestCasesFor200("/state")...),
		},
		{
			name:    "custom base path with trailing slash",
			handler: defaultDebugHandler,
			cfg: &configv1alpha1.HTTPDebugSpec{
				Enabled:  pointer.Bool(true),
				BasePath: "/state/",
			},
			cases: append(makeTestCasesFor404(), makeTestCasesFor200("/state")...),
		},
		{
			name: "error getting dynamic configs",
			handler: &mockDebugHandler{
				bootstrapConfig:    mockBootstrapConfig,
				dynamicConfig:      mockDynamicConfig,
				dynamicConfigError: assert.AnError,
			},
			cfg: &configv1alpha1.HTTPDebugSpec{
				Enabled: pointer.Bool(true),
			},
			cases: []*httphandlertesting.TestCase{
				{
					Name:     "error while getting dynamic config",
					Path:     "/debug/config/dynamic",
					WantCode: 500,
					AssertBody: testutils.AssertValueAll(
						testutils.AssertValueJSONEq(map[string]any{"error": assert.AnError.Error()}),
					),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			suite := httphandlertesting.NewSuite(&httphandlertesting.TestSuite{
				Method: http.MethodGet,
			})
			httphandler.ServeDebug(suite.GetMux(), tt.cfg, tt.handler)
			suite.Run(t, tt.cases)
		})
	}
}
