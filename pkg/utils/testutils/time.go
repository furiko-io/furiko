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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mktime parses a RFC3339 string or panics and returns a time.Time.
func Mktime(value string) time.Time {
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err) // panic ok for tests
	}
	return ts
}

// Mktimep parses a RFC3339 string or panics and returns a *time.Time.
func Mktimep(value string) *time.Time {
	t := Mktime(value)
	return &t
}

// Mkmtime parses a RFC3339 string or panics and returns a metav1.Time.
func Mkmtime(value string) metav1.Time {
	return metav1.NewTime(Mktime(value))
}

// Mkmtimep parses a RFC3339 string or panics and returns a *metav1.Time.
func Mkmtimep(value string) *metav1.Time {
	mt := Mkmtime(value)
	return &mt
}
