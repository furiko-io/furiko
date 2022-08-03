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

// IsFunc is a function that tests whether the target error is a certain error.
type IsFunc = func(target error) bool

// IsAny returns true if target matches any of the IsFunc functions.
// Always returns false if target is nil.
func IsAny(target error, errorIsFuncs ...IsFunc) bool {
	if target == nil {
		return false
	}
	for _, is := range errorIsFuncs {
		if is(target) {
			return true
		}
	}
	return false
}
