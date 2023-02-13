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

package testutils

import (
	"encoding/json"

	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// AssertErrorIs returns assert.ErrorAssertionFunc that asserts that the error
// is target.
func AssertErrorIs(target error) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		return assert.ErrorIs(t, err, target, i...)
	}
}

// AssertErrorIsNotFound returns assert.ErrorAssertionFunc that asserts that the error
// is a NotFoundError.
func AssertErrorIsNotFound() assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		return assert.True(t, apierrors.IsNotFound(err), i...)
	}
}

// AssertErrorContains returns assert.ErrorAssertionFunc that asserts that the error
// message contains str.
func AssertErrorContains(str string) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		return assert.Contains(t, err.Error(), str, i...)
	}
}

// AssertValueContains returns assert.ValueAssertionFunc that asserts that the value contains the given contents.
func AssertValueContains(contains interface{}) assert.ValueAssertionFunc {
	return func(t assert.TestingT, val interface{}, i ...interface{}) bool {
		return assert.Contains(t, val, contains, i...)
	}
}

// AssertValueJSONEq returns assert.ValueAssertionFunc that asserts that two JSON strings are equivalent.
func AssertValueJSONEq(expected any) assert.ValueAssertionFunc {
	return func(t assert.TestingT, val interface{}, i ...interface{}) bool {
		actual, ok := val.(string)
		if !ok {
			t.Errorf("wanted a string, got %T", val)
			return false
		}
		expectedBytes, err := json.Marshal(expected)
		if err != nil {
			t.Errorf("Marshal() got error: %v", err)
			return false
		}
		return assert.JSONEq(t, string(expectedBytes), actual, i...)
	}
}

// AssertValueAll composes multiple assert.ValueAssertionFunc together.
func AssertValueAll(fns ...assert.ValueAssertionFunc) assert.ValueAssertionFunc {
	return func(t assert.TestingT, val interface{}, i ...interface{}) bool {
		for _, fn := range fns {
			if !fn(t, val, i...) {
				return false
			}
		}
		return true
	}
}

// WantError checks err against assert.ErrorAssertionFunc, returning true if an
// error was encountered for short-circuiting.
func WantError(t assert.TestingT, wantErr assert.ErrorAssertionFunc, err error, i ...interface{}) bool {
	if wantErr == nil {
		wantErr = assert.NoError
	}
	wantErr(t, err, i...)
	return err != nil
}
