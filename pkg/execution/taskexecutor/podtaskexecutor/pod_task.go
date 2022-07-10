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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
)

const (
	reasonOOMKilled        = "OOMKilled"
	reasonError            = "Error"
	reasonDeadlineExceeded = "DeadlineExceeded"
)

// PodTask is a wrapper around Pod that fulfils Task.
type PodTask struct {
	*corev1.Pod
	client v1.PodInterface
}

var _ tasks.Task = (*PodTask)(nil)

func NewPodTask(pod *corev1.Pod, client v1.PodInterface) *PodTask {
	return &PodTask{Pod: pod, client: client}
}

func (p *PodTask) GetTaskRef() execution.TaskRef {
	reason, message := p.GetReasonMessage()
	task := execution.TaskRef{
		Name:              p.GetName(),
		CreationTimestamp: p.GetCreationTimestamp(),
		Status: execution.TaskStatus{
			State:   p.GetState(),
			Result:  p.GetResult(),
			Reason:  reason,
			Message: message,
		},
		NodeName:        p.Spec.NodeName,
		ContainerStates: p.GetContainerStates(),
	}

	if index, ok := p.GetRetryIndex(); ok {
		task.RetryIndex = index
	}
	if index, ok := p.GetParallelIndex(); ok {
		task.ParallelIndex = index
	}
	if t := p.GetRunningTimestamp(); !t.IsZero() {
		task.RunningTimestamp = &t
	}
	if t := p.GetFinishTimestamp(); !t.IsZero() {
		task.FinishTimestamp = &t
	}

	return task
}

func (p *PodTask) GetKind() string {
	return Kind
}

func (p *PodTask) GetRetryIndex() (int64, bool) {
	val, ok := p.Pod.Labels[tasks.LabelKeyTaskRetryIndex]
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}
	return int64(i), true
}

func (p *PodTask) GetParallelIndex() (*execution.ParallelIndex, bool) {
	val, ok := p.Annotations[tasks.AnnotationKeyTaskParallelIndex]
	if !ok {
		return nil, false
	}
	res := &execution.ParallelIndex{}
	if err := json.Unmarshal([]byte(val), res); err != nil {
		return nil, false
	}
	return res, true
}

func (p *PodTask) GetState() execution.TaskState {
	if !p.GetDeletionTimestamp().IsZero() && !p.IsFinished() {
		return execution.TaskKilling
	}

	switch p.Status.Phase {
	case corev1.PodRunning:
		return execution.TaskRunning
	case corev1.PodSucceeded, corev1.PodFailed:
		return execution.TaskTerminated
	default:
		return execution.TaskStarting
	}
}

func (p *PodTask) GetResult() execution.TaskResult {
	// Pod was OOMKilled, always use task failed.
	if p.IsOOMKilled() {
		return execution.TaskFailed
	}

	// Fallback to Pod phase for normal results.
	switch p.Status.Phase {
	case corev1.PodSucceeded:
		return execution.TaskSucceeded
	case corev1.PodFailed:
		return execution.TaskFailed
	case corev1.PodPending, corev1.PodRunning:
	}

	return ""
}

func (p *PodTask) GetRunningTimestamp() metav1.Time {
	return GetContainerStartTime(p.Pod)
}

func (p *PodTask) GetFinishTimestamp() metav1.Time {
	// If pod phase is not terminal, return zero.
	if !p.IsFinished() {
		return metav1.Time{}
	}

	// Try container termination time.
	if t := GetContainerTerminateTime(p.Pod); !t.IsZero() {
		return t
	}

	// If pod was DeadlineExceeded, container will not have Terminated state.
	// Use active deadline to compute the time it met the deadline.
	// This is only accurate if we actually set the active deadline relative to its start time.
	if p.Status.Reason == reasonDeadlineExceeded && p.Spec.ActiveDeadlineSeconds != nil {
		duration := time.Duration(*p.Spec.ActiveDeadlineSeconds) * time.Second
		return metav1.NewTime(p.Status.StartTime.Add(duration))
	}

	// Otherwise, we cannot determine finish time accurately.
	var fallbackFinishTime metav1.Time

	// Default to pod's start time if set.
	if p.Status.StartTime != nil {
		fallbackFinishTime = *p.Status.StartTime
	} else {
		// Last resort, use creation timestamp.
		fallbackFinishTime = p.GetCreationTimestamp()
	}

	return fallbackFinishTime
}

func (p *PodTask) IsFinished() bool {
	return p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed
}

func (p *PodTask) IsOOMKilled() bool {
	for _, container := range p.Status.ContainerStatuses {
		container := container
		if status := GetTerminationStatus(&container); status != nil && status.Reason == reasonOOMKilled {
			return true
		}
	}
	return false
}

func (p *PodTask) GetContainerStates() []execution.TaskContainerState {
	states := make([]execution.TaskContainerState, 0, len(p.Status.ContainerStatuses))
	for _, container := range p.Status.ContainerStatuses {
		var state execution.TaskContainerState
		state.ContainerID = container.ContainerID
		if status := container.State.Terminated; status != nil {
			state.ExitCode = status.ExitCode
			state.Signal = status.Signal
			state.Message = status.Message
			state.Reason = status.Reason
		}
		states = append(states, state)
	}
	return states
}

func (p *PodTask) GetReasonMessage() (string, string) { // nolint:gocognit
	// Take from Pod if exists.
	if p.Status.Reason != "" && p.Status.Message != "" {
		return p.Status.Reason, p.Status.Message
	}

	// Get failed scheduling reason.
	if condition := GetPodConditionScheduled(p.Pod); condition != nil &&
		condition.Status == corev1.ConditionFalse && hasReasonMessage(condition.Reason, condition.Message) {
		return condition.Reason, condition.Message
	}

	// Get failed pod initialization reason.
	if condition := GetPodConditionInitialized(p.Pod); condition != nil &&
		condition.Status == corev1.ConditionFalse && hasReasonMessage(condition.Reason, condition.Message) {
		return condition.Reason, condition.Message
	}

	containerStatusLists := [][]corev1.ContainerStatus{
		p.Status.ContainerStatuses,
		p.Status.InitContainerStatuses,
	}

	// Take from container Waiting state.
	for _, statuses := range containerStatusLists {
		for _, container := range statuses {
			if state := container.State.Waiting; state != nil {
				if hasReasonMessage(state.Reason, state.Message) {
					return state.Reason, state.Message
				}
			}
		}
	}

	// Take from container Terminated state.
	for _, statuses := range containerStatusLists {
		for _, container := range statuses {
			if state := container.State.Terminated; state != nil {
				message := state.Message
				if message == "" {
					message = p.getDefaultMessageForTermination(state)
				}
				if hasReasonMessage(state.Reason, message) {
					return state.Reason, message
				}
			}
		}
	}

	return "", ""
}

func hasReasonMessage(reason, message string) bool {
	return reason != "" && message != ""
}

// getDefaultMessageForTermination creates a default message for the terminated
// status of a pod.
func (p *PodTask) getDefaultMessageForTermination(status *corev1.ContainerStateTerminated) string {
	switch status.Reason {
	case reasonOOMKilled, reasonError:
		return fmt.Sprintf("Container exited with status %v", status.ExitCode)
	}

	return ""
}
