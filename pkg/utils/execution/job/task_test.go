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

package job_test

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

const (
	mockCreateTime = "2021-02-09T04:06:00Z"
	mockStartTime  = "2021-02-09T04:06:09Z"
	mockFinishTime = "2021-02-09T04:06:21Z"
	mockKillTime   = "2021-02-09T04:07:57Z"
)

var (
	stdCreateTime, _ = time.Parse(time.RFC3339, mockCreateTime)
	stdStartTime, _  = time.Parse(time.RFC3339, mockStartTime)
	stdFinishTime, _ = time.Parse(time.RFC3339, mockFinishTime)
	stdKillTime, _   = time.Parse(time.RFC3339, mockKillTime)
	createTime       = metav1.NewTime(stdCreateTime)
	createTime2      = metav1.NewTime(createTime.Add(time.Minute))
	startTime        = metav1.NewTime(stdStartTime)
	finishTime       = metav1.NewTime(stdFinishTime)
	finishTime2      = metav1.NewTime(stdFinishTime.Add(time.Minute))
	killTime         = metav1.NewTime(stdKillTime)
	one              = int32(1)
	two              = int32(2)
)

type stubTask struct {
	taskRef                        v1alpha1.TaskRef
	killTimestamp                  *metav1.Time
	deletionTimestamp              *metav1.Time
	retryIndex                     int64
	killedFromPendingTimeoutMarker bool
	killable                       bool
}

func (t *stubTask) GetName() string {
	return t.taskRef.Name
}

func (t *stubTask) GetTaskRef() v1alpha1.TaskRef {
	return t.taskRef
}

func (t *stubTask) GetKilledFromPendingTimeoutMarker() bool {
	return t.killedFromPendingTimeoutMarker
}

func (t *stubTask) SetKilledFromPendingTimeoutMarker(ctx context.Context) error {
	t.killedFromPendingTimeoutMarker = true
	return nil
}

func (t *stubTask) GetKind() string {
	return "Stub"
}

func (t *stubTask) GetRetryIndex() (int64, bool) {
	return t.retryIndex, true
}

func (t *stubTask) RequiresKillWithDeletion() bool {
	return t.killable
}

func (t *stubTask) GetDeletionTimestamp() *metav1.Time {
	return t.deletionTimestamp
}

func (t *stubTask) GetKillTimestamp() *metav1.Time {
	return t.killTimestamp
}

func (t *stubTask) SetKillTimestamp(ctx context.Context, ts time.Time) error {
	mts := metav1.NewTime(ts)
	t.killTimestamp = &mts
	return nil
}

func mkint64p(i int64) *int64 {
	return &i
}
