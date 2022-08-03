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

package errors

import (
	"fmt"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
)

// AssertIs returns assert.ErrorAssertionFunc that asserts that the error
// is target.
func AssertIs(target error) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		return assert.ErrorIs(t, err, target, i...)
	}
}

// AssertIsNotFound returns assert.ErrorAssertionFunc that asserts that the error
// is a NotFoundError.
func AssertIsNotFound() assert.ErrorAssertionFunc {
	return AssertIsFunc("NotFound", errors.IsNotFound)
}

// AssertIsForbidden returns assert.ErrorAssertionFunc that asserts that the error
// is a ForbiddenError.
func AssertIsForbidden() assert.ErrorAssertionFunc {
	return AssertIsFunc("Forbidden", errors.IsForbidden)
}

// AssertIsFunc returns assert.ErrorAssertionFunc that asserts that the
// error is a given error, tested using a custom errorIs function.
func AssertIsFunc(name string, errorIs IsFunc) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		return assert.True(t, errorIs(err),
			appendMsgAndArgs(i, "wanted %v, got %v", name, err)...)
	}
}

// AssertNotIsFunc returns assert.ErrorAssertionFunc that asserts that the
// error is not a given error, tested using a custom errorIs function.
func AssertNotIsFunc(name string, errorIs IsFunc) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		return assert.False(t, errorIs(err),
			appendMsgAndArgs(i, "error should not be %v, got %v", name, err)...)
	}
}

// AssertAll composes multiple assert.ErrorAssertionFunc together.
func AssertAll(fns ...assert.ErrorAssertionFunc) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		for _, fn := range fns {
			if fn(t, err, i...) {
				return true
			}
		}
		return false
	}
}

// appendMsgAndArgs appends a secondary message and args to msgAndArgs.
// If msgAndArgs is empty, then toAppend will be primarily used instead.
func appendMsgAndArgs(msgAndArgs []interface{}, toAppend ...interface{}) []interface{} {
	if len(msgAndArgs) > 0 {
		return []interface{}{fmt.Sprintf(
			"%v: %v",
			messageFromMsgAndArgs(msgAndArgs...),
			messageFromMsgAndArgs(toAppend...),
		)}
	}
	return toAppend
}

// Copied from github.com/stretchr/testify/assert.
func messageFromMsgAndArgs(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 || msgAndArgs == nil {
		return ""
	}
	if len(msgAndArgs) == 1 {
		msg := msgAndArgs[0]
		if msgAsStr, ok := msg.(string); ok {
			return msgAsStr
		}
		return fmt.Sprintf("%+v", msg)
	}
	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
	return ""
}

// Want checks err against assert.ErrorAssertionFunc, returning true if an
// error was encountered for short-circuiting.
func Want(t assert.TestingT, wantErr assert.ErrorAssertionFunc, err error, i ...interface{}) bool {
	if wantErr == nil {
		wantErr = assert.NoError
	}
	wantErr(t, err, i...)
	return err != nil
}
