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

package podtaskexecutor_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
)

func TestPodTask_GetState(t *testing.T) {
	tests := []struct {
		name string
		Pod  corev1.Pod
		want execution.TaskState
	}{
		{
			name: "pod created",
			Pod:  podCreated,
			want: execution.TaskStarting,
		},
		{
			name: "pod pending",
			Pod:  podPending,
			want: execution.TaskStarting,
		},
		{
			name: "pod container creating",
			Pod:  podContainerCreating,
			want: execution.TaskStarting,
		},
		{
			name: "pod running",
			Pod:  podRunning,
			want: execution.TaskRunning,
		},
		{
			name: "pod completed",
			Pod:  podCompleted,
			want: execution.TaskTerminated,
		},
		{
			name: "pod error",
			Pod:  podError,
			want: execution.TaskTerminated,
		},
		{
			name: "pod OOMKilled",
			Pod:  podOOMKilled,
			want: execution.TaskTerminated,
		},
		{
			name: "pod killing",
			Pod:  podKilling,
			want: execution.TaskKilling,
		},
		{
			name: "pod killed",
			Pod:  podKilled,
			want: execution.TaskTerminated,
		},
		{
			name: "pod killed from pending timeout",
			Pod:  podKilledByPendingTimeout,
			want: execution.TaskTerminated,
		},
		{
			name: "pod DeadlineExceeded",
			Pod:  podDeadlineExceeded,
			want: execution.TaskTerminated,
		},
		// TODO(irvinlim): Disable these test cases until we make a decision for https://github.com/furiko-io/furiko/issues/64
		// {
		// 	name: "pod had PodPending and DeadlineExceeded",
		// 	Pod:  podPendingDeadlineExceeded,
		// 	want: execution.TaskTerminated,
		// },
		// {
		// 	name: "pod had DeadlineExceeded with kill timestamp",
		// 	Pod:  podDeadlineExceededWithKillTimestamp,
		// 	want: execution.TaskTerminated,
		// },
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := podtaskexecutor.NewPodTask(&tt.Pod, nil)
			if got := p.GetState(); got != tt.want {
				t.Errorf("GetState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodTask_GetResult(t *testing.T) {
	tests := []struct {
		name string
		Pod  corev1.Pod
		want execution.TaskResult
	}{
		{
			name: "no result",
			Pod:  podPending,
		},
		{
			name: "Success",
			Pod:  podCompleted,
			want: execution.TaskSucceeded,
		},
		{
			name: "Error",
			Pod:  podError,
			want: execution.TaskFailed,
		},
		{
			name: "OOMKilled - Container exited with status 0",
			Pod:  podOOMKilledWithExitCode0,
			want: execution.TaskFailed,
		},
		{
			name: "OOMKilled - Container exited with status 137",
			Pod:  podOOMKilled,
			want: execution.TaskFailed,
		},
		{
			name: "DeadlineExceeded",
			Pod:  podDeadlineExceeded,
			want: execution.TaskDeadlineExceeded,
		},
		{
			name: "Killed",
			Pod:  podDeadlineExceededWithKillTimestamp,
			want: execution.TaskKilled,
		},
		{
			name: "PodPending with DeadlineExceeded",
			Pod:  podPendingDeadlineExceeded,
			want: execution.TaskDeadlineExceeded,
		},
		{
			name: "PendingTimeout",
			Pod:  podKilledByPendingTimeout,
			want: execution.TaskPendingTimeout,
		},
		{
			name: "Killed by pending timeout, still running",
			Pod:  podKillingByPendingTimeout,
		},
		{
			name: "Killed by pending timeout, but succeeded",
			Pod:  podSucceededButKillByPendingTimeout,
			want: execution.TaskSucceeded,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := podtaskexecutor.NewPodTask(&tt.Pod, nil)
			got := p.GetResult()
			if got != tt.want {
				t.Errorf("GetResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodTask_GetRunningTimestamp(t *testing.T) {
	tests := []struct {
		name string
		Pod  corev1.Pod
		want metav1.Time
	}{
		{
			name: "Pending",
			Pod:  podPending,
			want: metav1.Time{},
		},
		{
			name: "Running",
			Pod:  podRunning,
			want: containerStartTime,
		},
		{
			name: "Completed",
			Pod:  podCompleted,
			want: containerStartTime,
		},
		{
			name: "already terminated with zero start time",
			Pod:  podCompletedWithZeroTime,
			want: metav1.Time{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := podtaskexecutor.NewPodTask(&tt.Pod, nil)
			got := p.GetRunningTimestamp()
			if !tt.want.Equal(&got) {
				t.Errorf("GetRunningTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodTask_GetFinishTimestamp(t *testing.T) {
	tests := []struct {
		name string
		Pod  corev1.Pod
		want metav1.Time
	}{
		{
			name: "Pending",
			Pod:  podPending,
			want: metav1.Time{},
		},
		{
			name: "Running",
			Pod:  podRunning,
			want: metav1.Time{},
		},
		{
			name: "Completed",
			Pod:  podCompleted,
			want: containerFinishTime,
		},
		{
			name: "Killed by pending timeout",
			Pod:  podKilledByPendingTimeout,
			want: killTime,
		},
		{
			name: "Zero terminated time",
			Pod:  podCompletedWithZeroTime,
			want: startTime,
		},
		{
			name: "Zero last termination state time",
			Pod:  podCompletedWithLastTerminationStateZeroTime,
			want: startTime,
		},
		{
			name: "Failed with no container status",
			Pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: createTime,
				},
				Status: corev1.PodStatus{
					Phase:     corev1.PodFailed,
					StartTime: &startTime,
				},
			},
			want: startTime,
		},
		{
			name: "Failed with no start time",
			Pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: createTime,
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
				},
			},
			want: createTime,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := podtaskexecutor.NewPodTask(&tt.Pod, nil)
			got := p.GetFinishTimestamp()
			if !tt.want.Equal(&got) {
				t.Errorf("GetFinishTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodTask_RequiresKillWithDeletion(t *testing.T) {
	tests := []struct {
		name string
		Pod  corev1.Pod
		want bool
	}{
		{
			name: "not yet scheduled",
			Pod:  podPendingUnschedulable,
			want: true,
		},
		{
			name: "already scheduled",
			Pod:  podPending,
			want: false,
		},
		{
			name: "already scheduled, cannot pull image",
			Pod:  podErrImagePull,
			want: false,
		},
		{
			name: "pod completed",
			Pod:  podCompleted,
			want: false,
		},
		{
			name: "pod error",
			Pod:  podError,
			want: false,
		},
		{
			name: "deadline exceeded",
			Pod:  podDeadlineExceeded,
			want: false,
		},
		{
			name: "deadline exceeded",
			Pod:  podPendingDeadlineExceeded,
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := podtaskexecutor.NewPodTask(&tt.Pod, nil)
			if got := p.RequiresKillWithDeletion(); got != tt.want {
				t.Errorf("RequiresKillWithDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodTask_GetReasonMessage(t *testing.T) {
	type reasonMessage struct {
		reason  string
		message string
	}
	tests := []struct {
		name string
		Pod  corev1.Pod
		want reasonMessage
	}{
		{
			name: "Pending",
			Pod:  podPending,
			want: reasonMessage{},
		},
		{
			name: "Running",
			Pod:  podRunning,
			want: reasonMessage{},
		},
		{
			name: "Completed",
			Pod:  podCompleted,
			want: reasonMessage{},
		},
		{
			name: "ImagePullBackOff",
			Pod:  podImagePullBackOff,
			want: reasonMessage{
				reason:  "ImagePullBackOff",
				message: "Back-off pulling image \"hello-world\"",
			},
		},
		{
			name: "ErrImagePull",
			Pod:  podErrImagePull,
			want: reasonMessage{
				reason: "ErrImagePull",
				message: "rpc error: code = Unknown desc = failed to resolve image " +
					"\"hello-world\": " +
					"no available registry endpoint: hello-world not found",
			},
		},
		{
			name: "Error",
			Pod:  podError,
			want: reasonMessage{
				reason:  "Error",
				message: "Container exited with status 255",
			},
		},
		{
			name: "OOMKilled",
			Pod:  podOOMKilled,
			want: reasonMessage{
				reason:  "OOMKilled",
				message: "Container exited with status 137",
			},
		},
		{
			name: "DeadlineExceeded",
			Pod:  podDeadlineExceeded,
			want: reasonMessage{
				reason:  "DeadlineExceeded",
				message: "Pod was active on the node longer than the specified deadline",
			},
		},
		{
			name: "Unschedulable",
			Pod:  podPendingUnschedulable,
			want: reasonMessage{
				reason:  "Unschedulable",
				message: "0/4 nodes are available: 4 Insufficient cpu.",
			},
		},
		{
			name: "ContainersNotInitialized",
			Pod:  podConditionInitializedFalse,
			want: reasonMessage{
				reason:  "ContainersNotInitialized",
				message: "containers with incomplete status: [init-container]",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := podtaskexecutor.NewPodTask(&tt.Pod, nil)
			reason, message := p.GetReasonMessage()
			if reason != tt.want.reason {
				t.Errorf("GetStatusMessage() reason = %v, want %v", reason, tt.want.reason)
			}
			if message != tt.want.message {
				t.Errorf("GetStatusMessage() message = %v, want %v", message, tt.want.message)
			}
		})
	}
}
