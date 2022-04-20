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
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/utils/k8sutils"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
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

	if t := p.GetRunningTimestamp(); !t.IsZero() {
		task.RunningTimestamp = &t
	}
	if t := p.GetFinishTimestamp(); !t.IsZero() {
		task.FinishTimestamp = &t
	}

	return task
}

func (p *PodTask) GetKind() string {
	return "Pod"
}

func (p *PodTask) GetRetryIndex() (int64, bool) {
	val, ok := p.Pod.Labels[LabelKeyTaskRetryIndex]
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}
	return int64(i), true
}

// RequiresKillWithDeletion returns true if the Task should be killed with
// deletion instead of active deadline. Currently, we only enforce deletion if
// the Pod is not yet scheduled, otherwise we should always use kill timestamp
// to allow for graceful termination.
func (p *PodTask) RequiresKillWithDeletion() bool {
	// If a Pod is not pending, it is definitely already scheduled.
	return p.Status.Phase == corev1.PodPending &&
		// Check the PodCondition, which may in some cases be non-existent from PodStatus
		!IsPodConditionScheduled(p.Pod) &&
		// Pod is not yet acknowledged by kubelet, which means it was not yet scheduled
		p.Status.StartTime.IsZero()
}

func (p *PodTask) GetKillTimestamp() *metav1.Time {
	if val, ok := p.Pod.Annotations[LabelKeyTaskKillTimestamp]; ok {
		if unix, err := strconv.Atoi(val); err == nil {
			t := metav1.Unix(int64(unix), 0)
			return &t
		}
	}
	return nil
}

func (p *PodTask) SetKillTimestamp(ctx context.Context, ts time.Time) error {
	newPod := p.Pod.DeepCopy()

	// Cannot increase kill timestamp further than it was before.
	if ktime.IsTimeSetAndEarlierThan(p.GetKillTimestamp(), ts) {
		ts = p.GetKillTimestamp().Time
	}

	// Add annotation for kill timestamp.
	// This will be the authoritative source of truth for the time we want to kill the pod.
	k8sutils.SetAnnotation(newPod, LabelKeyTaskKillTimestamp, strconv.Itoa(int(ts.Unix())))

	// Compute active deadline.
	var ads int64
	switch {
	case p.Status.StartTime == nil:
		// No start time. We will set active deadline to minimum.
		// When kubelet acknowledges the pod, it will immediately kill it after 1 second.
		ads = 1
	case ts.Before(p.Status.StartTime.Time):
		// For some reason we want to kill it before it started.
		ads = 1
	default:
		// Compute the duration between killTimestamp and startTime as the ADS.
		ads = int64(ts.Sub(p.Status.StartTime.Time).Seconds())
	}

	// Cannot set higher than previous value.
	if newPod.Spec.ActiveDeadlineSeconds != nil && ads > *newPod.Spec.ActiveDeadlineSeconds {
		ads = *newPod.Spec.ActiveDeadlineSeconds
	}

	// If value was set less than 1 for any other reason, round it up to 1 to conform to range.
	if ads < 1 {
		ads = 1
	}

	// Set active deadline.
	newPod.Spec.ActiveDeadlineSeconds = &ads

	updatedPod, err := p.client.Update(ctx, newPod, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not update pod")
	}

	p.Pod = updatedPod
	return nil
}

func (p *PodTask) GetKilledFromPendingTimeoutMarker() bool {
	_, ok := p.Pod.Annotations[LabelKeyKilledFromPendingTimeout]
	return ok
}

func (p *PodTask) SetKilledFromPendingTimeoutMarker(ctx context.Context) error {
	newPod := p.Pod.DeepCopy()
	k8sutils.SetAnnotation(newPod, LabelKeyKilledFromPendingTimeout, "1")

	updatedPod, err := p.client.Update(ctx, newPod, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not update pod")
	}

	p.Pod = updatedPod
	return nil
}

func (p *PodTask) GetState() execution.TaskState {
	// Pod has a kill timestamp in the past.
	if ktime.IsTimeSetAndEarlier(p.GetKillTimestamp()) {
		if !p.IsFinished() {
			// Pod in the midst of killing.
			return execution.TaskKilling
		} else if p.Status.Phase == corev1.PodFailed && p.IsDeadlineExceeded() {
			// Pod was killed using active deadline.
			return execution.TaskKilled
		}
	}

	// Pod was unschedulable (may have used active deadline).
	if p.IsKilledFromPendingTimeout() {
		return execution.TaskPendingTimeout
	}

	// Pod has DeadlineExceeded but no kill timestamp.
	if p.IsDeadlineExceeded() {
		return execution.TaskDeadlineExceeded
	}

	// Pod was OOMKilled, always use task failed. This is because kubelet may set
	// PodSucceeded phase if the exit code is 0 even though oom_killer was invoked.
	// TODO(irvinlim): This may not be necessary
	if p.IsOOMKilled() {
		return execution.TaskFailed
	}

	// Check pod phase
	switch p.Status.Phase {
	case corev1.PodPending:
		if len(p.Status.ContainerStatuses) > 0 {
			return execution.TaskStarting
		}
		return execution.TaskStaging
	case corev1.PodRunning:
		return execution.TaskRunning
	case corev1.PodSucceeded:
		return execution.TaskSuccess
	case corev1.PodFailed:
		return execution.TaskFailed
	}

	return execution.TaskStaging
}

func (p *PodTask) GetResult() *execution.JobResult {
	// Pod was OOMKilled, always use task failed.
	if p.IsOOMKilled() {
		return job.GetResultPtr(execution.JobResultTaskFailed)
	}

	// Pod was killed by pending timeout.
	if p.IsKilledFromPendingTimeout() {
		return job.GetResultPtr(execution.JobResultPendingTimeout)
	}

	// Killed pod using active deadline.
	if p.IsDeadlineExceeded() {
		// Kill timestamp was set but not from pending timeout.
		if !p.GetKillTimestamp().IsZero() {
			return job.GetResultPtr(execution.JobResultKilled)
		}
		return job.GetResultPtr(execution.JobResultDeadlineExceeded)
	}

	// Fallback to Pod phase for normal results.
	switch p.Status.Phase {
	case corev1.PodSucceeded:
		return job.GetResultPtr(execution.JobResultSuccess)
	case corev1.PodFailed:
		return job.GetResultPtr(execution.JobResultTaskFailed)
	case corev1.PodPending, corev1.PodRunning:
	}

	return nil
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

func (p *PodTask) IsKilledFromPendingTimeout() bool {
	// Pod is not yet in failed state. It may be possible for a Pod to race between
	// succeeding and being killed by pending timeout.
	if p.Status.Phase != corev1.PodFailed {
		return false
	}
	return p.GetKilledFromPendingTimeoutMarker()
}

func (p *PodTask) IsDeadlineExceeded() bool {
	return p.Status.Reason == reasonDeadlineExceeded
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
