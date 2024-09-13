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

package podtaskexecutor

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

// IsPodConditionScheduled returns whether the pod has been scheduled.
// This does not necessarily mean that the pod is currently in a Scheduled state.
func IsPodConditionScheduled(pod *corev1.Pod) bool {
	condition := GetPodConditionScheduled(pod)
	if condition != nil && condition.Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// GetPodConditionScheduled returns the PodScheduled pod condition.
func GetPodConditionScheduled(pod *corev1.Pod) *corev1.PodCondition {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled {
			return &condition
		}
	}
	return nil
}

// GetPodConditionInitialized returns the PodInitialized pod condition.
func GetPodConditionInitialized(pod *corev1.Pod) *corev1.PodCondition {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodInitialized {
			return &condition
		}
	}
	return nil
}

// GetTerminationStatus returns either the current container's Terminated or LastTerminateState.
func GetTerminationStatus(container *corev1.ContainerStatus) *corev1.ContainerStateTerminated {
	if container.State.Terminated != nil {
		return container.State.Terminated
	}
	if container.LastTerminationState.Terminated != nil {
		return container.LastTerminationState.Terminated
	}
	return nil
}

// GetContainerStartTime returns the latest timestamp that any container started
// running.
func GetContainerStartTime(pod *corev1.Pod) metav1.Time {
	var t metav1.Time

	for _, container := range pod.Status.ContainerStatuses {
		// NOTE(irvinlim): Use IsUnixZero to handle case when kubelet reports
		// "1970-01-01T00:00:00Z", which is not a zero time.Time.
		if status := container.State.Running; status != nil && !ktime.IsUnixZero(&status.StartedAt) {
			t = *ktime.TimeMax(&status.StartedAt, &t)
		}
		if status := container.State.Terminated; status != nil && !ktime.IsUnixZero(&status.StartedAt) {
			t = *ktime.TimeMax(&status.StartedAt, &t)
		}
	}

	return t
}

// GetContainerTerminateTime returns the latest timestamp at which any container
// terminated.
func GetContainerTerminateTime(pod *corev1.Pod) metav1.Time {
	var t metav1.Time

	for _, container := range pod.Status.ContainerStatuses {
		if status := GetTerminationStatus(&container); status != nil && !ktime.IsUnixZero(&status.FinishedAt) {
			t = *ktime.TimeMax(&status.FinishedAt, &t)
		}
	}

	return t
}
