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

package testing

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Config contains common configuration for a single Suite.
type Config struct {
	// Defines the default path to request.
	Path string

	// Defines the default HTTP method to use.
	Method string
}

// Case represents a single test case.
type Case struct {
	// Name of the test case.
	Name string

	// Path to request.
	// If empty, uses the default path in the Config.
	Path string

	// Method to request.
	// If empty, uses the default method in the Config.
	Method string

	// Request body.
	Body []byte

	// WantError defines whether the HTTP request is expected to have a non-2xx response code.
	WantError bool

	// WantCode defines the HTTP code that is expected.
	// If not defined, then HTTP code checking is omitted.
	WantCode int

	// AssertBody validates the response body.
	AssertBody assert.ValueAssertionFunc
}

// Suite is a test suite for HTTP testing.
type Suite struct {
	mux   *http.ServeMux
	suite *Config
}

func NewSuite(test *Config) *Suite {
	return &Suite{
		mux:   http.NewServeMux(),
		suite: test,
	}
}

// GetMux returns the *http.ServeMux.
func (s *Suite) GetMux() *http.ServeMux {
	return s.mux
}

// Run the test suite.
func (s *Suite) Run(t *testing.T, cases []*Case) {
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			s.runTestCase(t, tt)
		})
	}
}

func (s *Suite) runTestCase(t *testing.T, tt *Case) {
	rec := httptest.NewRecorder()

	method := tt.Method
	if method == "" {
		method = s.suite.Method
	}
	path := tt.Path
	if path == "" {
		path = s.suite.Path
	}
	var body io.Reader
	if len(tt.Body) > 0 {
		body = bytes.NewBuffer(tt.Body)
	}

	req := httptest.NewRequest(method, path, body)
	s.GetMux().ServeHTTP(rec, req)

	resp := rec.Result()
	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	defer assert.NoError(t, resp.Body.Close())

	// Check the status code if defined.
	if tt.WantCode != 0 {
		assert.Equal(t, tt.WantCode, resp.StatusCode, "StatusCode not equal")
	}

	// Check WantError only if WantCode is not defined.
	if tt.WantCode == 0 {
		isError := resp.StatusCode >= 400
		assert.Equalf(t, tt.WantError, isError, "WantError = %v, got status code %v", tt.WantError, resp.StatusCode)
	}

	// Validate the response body.
	if tt.AssertBody != nil {
		tt.AssertBody(t, string(respBody), "Response body not equal, got: %v", string(respBody))
	}
}
