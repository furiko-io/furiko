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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	mockCreateTime          = "2021-02-09T04:06:00Z"
	mockStartTime           = "2021-02-09T04:06:09Z"
	mockContainerStartTime  = "2021-02-09T04:06:15Z"
	mockContainerFinishTime = "2021-02-09T04:06:21Z"
	mockKillTime            = "2021-02-09T04:08:09Z"

	containerName = "container"
	containerID   = "containerd://b39c8972a4030c99e30b434c6e865fa4f39d218bf086d5231823f3d56e1b45f4"
	image         = "hello-world"
)

var (
	stdCreateTime, _          = time.Parse(time.RFC3339, mockCreateTime)
	stdStartTime, _           = time.Parse(time.RFC3339, mockStartTime)
	stdContainerStartTime, _  = time.Parse(time.RFC3339, mockContainerStartTime)
	stdContainerFinishTime, _ = time.Parse(time.RFC3339, mockContainerFinishTime)
	stdKillTime, _            = time.Parse(time.RFC3339, mockKillTime)
	createTime                = metav1.NewTime(stdCreateTime)
	startTime                 = metav1.NewTime(stdStartTime)
	containerStartTime        = metav1.NewTime(stdContainerStartTime)
	containerFinishTime       = metav1.NewTime(stdContainerFinishTime)
	killTime                  = metav1.NewTime(stdKillTime)
	valTrue                   = true
	valFalse                  = false
)

var (
	conditionPodScheduled = corev1.PodCondition{
		Type:   corev1.PodScheduled,
		Status: corev1.ConditionTrue,
	}

	conditionPodInitialized = corev1.PodCondition{
		Type:   corev1.PodInitialized,
		Status: corev1.ConditionTrue,
	}

	conditionsPodScheduledAndInit = []corev1.PodCondition{
		conditionPodScheduled,
		conditionPodInitialized,
	}

	podCreated = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	// podPendingUnschedulable refers to a pod that is unschedulable by kube-scheduler.
	podPendingUnschedulable = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{
				{
					Type:    corev1.PodScheduled,
					Status:  corev1.ConditionFalse,
					Reason:  "Unschedulable",
					Message: "0/4 nodes are available: 4 Insufficient cpu.",
				},
			},
		},
	}

	// podPending refers to a pod that has been scheduled and picked up by kubelet.
	podPending = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodPending,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
		},
	}

	podContainerCreating = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodPending,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{},
					},
				},
			},
		},
	}

	podRunning = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:    containerName,
					Image:   image,
					Ready:   true,
					Started: &valTrue,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: containerStartTime,
						},
					},
				},
			},
		},
	}

	podCompleted = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodSucceeded,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ContainerID: containerID,
							FinishedAt:  containerFinishTime,
							Reason:      "Completed",
							StartedAt:   containerStartTime,
						},
					},
				},
			},
		},
	}

	podCompletedWithZeroTime = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodSucceeded,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ContainerID: containerID,
							FinishedAt:  metav1.Time{},
							StartedAt:   metav1.Time{},
						},
					},
				},
			},
		},
	}

	podCompletedWithLastTerminationStateZeroTime = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodSucceeded,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:        containerName,
					ContainerID: containerID,
					Image:       image,
					Ready:       false,
					Started:     &valFalse,
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ContainerID: containerID,
							FinishedAt:  metav1.Time{},
							StartedAt:   metav1.Time{},
						},
					},
				},
			},
		},
	}

	podImagePullBackOff = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodPending,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Message: "Back-off pulling image \"" + image + "\"",
							Reason:  "ImagePullBackOff",
						},
					},
				},
			},
		},
	}

	podErrImagePull = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodPending,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Message: "rpc error: code = Unknown desc = failed to resolve image " +
								"\"" + image + "\": " +
								"no available registry endpoint: " + image + " not found",
							Reason: "ErrImagePull",
						},
					},
				},
			},
		},
	}

	podError = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodFailed,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ContainerID: containerID,
							ExitCode:    255,
							FinishedAt:  containerFinishTime,
							Reason:      "Error",
							StartedAt:   containerStartTime,
						},
					},
				},
			},
		},
	}

	podOOMKilled = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodFailed,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ContainerID: containerID,
							FinishedAt:  containerFinishTime,
							ExitCode:    137,
							Reason:      "OOMKilled",
							StartedAt:   containerStartTime,
						},
					},
				},
			},
		},
	}

	podOOMKilledWithExitCode0 = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodSucceeded,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ContainerID: containerID,
							FinishedAt:  containerFinishTime,
							Reason:      "OOMKilled",
							ExitCode:    0,
							StartedAt:   containerStartTime,
						},
					},
				},
			},
		},
	}

	// DeadlineExceeded pods may not have conditions set.
	podDeadlineExceeded = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:     corev1.PodFailed,
			StartTime: &startTime,
			Reason:    "DeadlineExceeded",
			Message:   "Pod was active on the node longer than the specified deadline",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ContainerID: containerID,
							FinishedAt:  containerFinishTime,
							StartedAt:   containerStartTime,
							Reason:      "Completed",
						},
					},
				},
			},
		},
	}

	podDeleting = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
			DeletionTimestamp: &killTime,
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			StartTime:  &startTime,
			Conditions: conditionsPodScheduledAndInit,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Image: image,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: containerStartTime,
						},
					},
				},
			},
		},
	}

	podConditionInitializedFalse = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: createTime,
		},
		Status: corev1.PodStatus{
			Phase:     corev1.PodFailed,
			StartTime: &startTime,
			Conditions: []corev1.PodCondition{
				conditionPodScheduled,
				{
					Type:    corev1.PodInitialized,
					Status:  corev1.ConditionFalse,
					Reason:  "ContainersNotInitialized",
					Message: "containers with incomplete status: [init-container]",
				},
			},
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "init-container",
					Image: image,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ContainerID: containerID,
							FinishedAt:  containerFinishTime,
							StartedAt:   containerStartTime,
							ExitCode:    1,
							Reason:      "Error",
						},
					},
				},
			},
		},
	}
)
