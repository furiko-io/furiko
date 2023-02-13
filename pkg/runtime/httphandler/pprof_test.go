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

	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/httphandler"
	httphandlertesting "github.com/furiko-io/furiko/pkg/runtime/httphandler/testing"
)

func TestServePprof(t *testing.T) {
	type testCase struct {
		name  string
		cfg   *configv1alpha1.HTTPPprofSpec
		cases []*httphandlertesting.TestCase
	}
	for _, tt := range []testCase{
		{
			name: "default config",
			cases: []*httphandlertesting.TestCase{
				{
					Name:     "should not be enabled",
					Path:     "/debug/pprof/",
					WantCode: 404,
				},
			},
		},
		{
			name: "explicitly disabled",
			cfg: &configv1alpha1.HTTPPprofSpec{
				Enabled: pointer.Bool(false),
			},
			cases: []*httphandlertesting.TestCase{
				{
					Name:     "should not be enabled",
					Path:     "/debug/pprof/",
					WantCode: 404,
				},
			},
		},
		{
			name: "config is enabled",
			cfg: &configv1alpha1.HTTPPprofSpec{
				Enabled: pointer.Bool(true),
			},
			cases: []*httphandlertesting.TestCase{
				{
					Name:     "should have redirect",
					Path:     "/debug/pprof",
					WantCode: 301,
				},
				{
					Name:     "should be ok",
					Path:     "/debug/pprof/",
					WantCode: 200,
				},
				{
					Name:     "can load goroutine",
					Path:     "/debug/pprof/goroutine",
					WantCode: 200,
				},
				{
					Name:     "cannot load invalid profile",
					Path:     "/debug/pprof/invalid",
					WantCode: 404,
				},
			},
		},
		{
			name: "custom path",
			cfg: &configv1alpha1.HTTPPprofSpec{
				Enabled:   pointer.Bool(true),
				IndexPath: "/_internal/debug/pprof",
			},
			cases: []*httphandlertesting.TestCase{
				{
					Name:     "default route is not registered",
					Path:     "/debug/pprof/",
					WantCode: 404,
				},
				{
					Name:     "should have redirect",
					Path:     "/_internal/debug/pprof",
					WantCode: 301,
				},
				{
					Name:     "should be ok",
					Path:     "/_internal/debug/pprof/",
					WantCode: 200,
				},
				{
					Name:     "can load goroutine",
					Path:     "/_internal/debug/pprof/goroutine",
					WantCode: 200,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			suite := httphandlertesting.NewSuite(&httphandlertesting.TestSuite{
				Method: http.MethodGet,
			})
			httphandler.ServePprof(suite.GetMux(), tt.cfg)
			suite.Run(t, tt.cases)
		})
	}
}
