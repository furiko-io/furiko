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

package time_test

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/furiko-io/furiko/pkg/utils/testutils"
	timeutil "github.com/furiko-io/furiko/pkg/utils/time"
)

func TestTime(t *testing.T) {
	type testCase struct {
		name string
		a    time.Time
		b    time.Time
		want time.Time
	}
	tests := []struct {
		f     func(a, b time.Time) time.Time
		cases []testCase
	}{
		{
			f: timeutil.Max,
			cases: []testCase{

				{
					name: "zero times",
					want: time.Time{},
				},
				{
					name: "a is zero time",
					a:    time.Time{},
					b:    testutils.Mktime("2022-01-01T00:00:00Z"),
					want: testutils.Mktime("2022-01-01T00:00:00Z"),
				},
				{
					name: "b is zero time",
					a:    testutils.Mktime("2022-01-01T00:00:00Z"),
					b:    time.Time{},
					want: testutils.Mktime("2022-01-01T00:00:00Z"),
				},
				{
					name: "get max from a",
					a:    testutils.Mktime("2022-01-02T00:00:00Z"),
					b:    testutils.Mktime("2022-01-01T00:00:00Z"),
					want: testutils.Mktime("2022-01-02T00:00:00Z"),
				},
				{
					name: "get max from b",
					a:    testutils.Mktime("2022-01-01T00:00:00Z"),
					b:    testutils.Mktime("2022-01-02T00:00:00Z"),
					want: testutils.Mktime("2022-01-02T00:00:00Z"),
				},
			},
		},
		{
			f: timeutil.Min,
			cases: []testCase{

				{
					name: "zero times",
					want: time.Time{},
				},
				{
					name: "a is zero time",
					a:    time.Time{},
					b:    testutils.Mktime("2022-01-01T00:00:00Z"),
					want: time.Time{},
				},
				{
					name: "b is zero time",
					a:    testutils.Mktime("2022-01-01T00:00:00Z"),
					b:    time.Time{},
					want: time.Time{},
				},
				{
					name: "get min from a",
					a:    testutils.Mktime("2022-01-01T00:00:00Z"),
					b:    testutils.Mktime("2022-01-02T00:00:00Z"),
					want: testutils.Mktime("2022-01-01T00:00:00Z"),
				},
				{
					name: "get min from b",
					a:    testutils.Mktime("2022-01-02T00:00:00Z"),
					b:    testutils.Mktime("2022-01-01T00:00:00Z"),
					want: testutils.Mktime("2022-01-01T00:00:00Z"),
				},
			},
		},
		{
			f: timeutil.MinNonZero,
			cases: []testCase{

				{
					name: "zero times",
					want: time.Time{},
				},
				{
					name: "a is zero time",
					a:    time.Time{},
					b:    testutils.Mktime("2022-01-01T00:00:00Z"),
					want: testutils.Mktime("2022-01-01T00:00:00Z"),
				},
				{
					name: "b is zero time",
					a:    testutils.Mktime("2022-01-01T00:00:00Z"),
					b:    time.Time{},
					want: testutils.Mktime("2022-01-01T00:00:00Z"),
				},
				{
					name: "get min from a",
					a:    testutils.Mktime("2022-01-01T00:00:00Z"),
					b:    testutils.Mktime("2022-01-02T00:00:00Z"),
					want: testutils.Mktime("2022-01-01T00:00:00Z"),
				},
				{
					name: "get min from b",
					a:    testutils.Mktime("2022-01-02T00:00:00Z"),
					b:    testutils.Mktime("2022-01-01T00:00:00Z"),
					want: testutils.Mktime("2022-01-01T00:00:00Z"),
				},
			},
		},
	}
	for _, funcTest := range tests {
		funcName := getFunctionName(funcTest.f)
		t.Run(funcName, func(t *testing.T) {
			for _, tt := range funcTest.cases {
				t.Run(tt.name, func(t *testing.T) {
					if got := funcTest.f(tt.a, tt.b); !reflect.DeepEqual(got, tt.want) {
						t.Errorf("%v() = %v, want %v", funcName, got, tt.want)
					}
				})
			}
		})
	}
}

func getFunctionName(i interface{}) string {
	// Get the full function name.
	name := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()

	// Remove the package URL (like github.com/furiko-io/furiko)
	name = getLastSplit(name, "/")

	// Remove the package name (like time)
	name = getLastSplit(name, ".")

	return name
}

func getLastSplit(s, sep string) string {
	split := strings.Split(s, sep)
	if len(split) == 0 {
		return ""
	}
	return split[len(split)-1]
}
