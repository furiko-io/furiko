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

package croncontroller_test

import (
	"context"
	"time"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/croncontroller"
)

type fakeRecorder struct {
	Events []string
}

var _ croncontroller.Recorder = (*fakeRecorder)(nil)

func newFakeRecorder() *fakeRecorder {
	return &fakeRecorder{}
}

func (f *fakeRecorder) CreatedJob(_ context.Context, _ *execution.JobConfig, _ *execution.Job) {
	f.addEvent("CreatedJob")
}

func (f *fakeRecorder) CreateJobFailed(_ context.Context, _ *execution.JobConfig, _ *execution.Job, _ string) {
	f.addEvent("CreateJobFailed")
}

func (f *fakeRecorder) SkippedJobSchedule(_ context.Context, _ *execution.JobConfig, _ time.Time, _ string) {
	f.addEvent("SkippedJobSchedule")
}

func (f *fakeRecorder) addEvent(event string) {
	f.Events = append(f.Events, event)
}

func (f *fakeRecorder) Clear() {
	f.Events = nil
}
