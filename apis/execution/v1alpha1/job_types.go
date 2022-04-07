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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobSpec defines the desired state of a Job.
type JobSpec struct {
	// ConfigName allows specifying the name of the JobConfig to create the Job
	// from. The JobConfig must be in the same namespace as the Job.
	//
	// It is provided as a write-only input field for convenience, and will override
	// the template, labels and annotations from the JobConfig's template.
	//
	// This field will never be returned from the API. To look up the parent
	// JobConfig, use ownerReferences.
	//
	// +optional
	ConfigName string `json:"configName,omitempty"`

	// Specifies the type of Job.
	// Can be one of: Adhoc, Scheduled
	//
	// Default: Adhoc
	// +optional
	Type JobType `json:"type"`

	// Specifies optional start policy for a Job, which specifies certain conditions
	// which have to be met before a Job is started.
	//
	// +optional
	StartPolicy *StartPolicySpec `json:"startPolicy,omitempty"`

	// Template specifies how to create the Job.
	// +optional
	Template *JobTemplateSpec `json:"template,omitempty"`

	// Specifies key-values pairs of values for Options, in JSON or YAML format.
	//
	// Example specification:
	//
	//   spec:
	//     optionValues: |-
	//       myStringOption: "value"
	//       myBoolOption: true
	//       mySelectOption:
	//       - option1
	//       - option3
	//
	// Each entry in the optionValues struct should consist of the option's name,
	// and the value could be an arbitrary type that corresponds to the option's
	// type itself. Each option value specified will be evaluated to a string based
	// on the JobConfig's OptionsSpec and added to Substitutions. If the key also
	// exists in Substitutions, that one takes priority.
	//
	// Cannot be updated after creation.
	//
	// +optional
	OptionValues string `json:"optionValues,omitempty"`

	// Defines key-value pairs of context variables to be substituted into the
	// TaskTemplate. Each entry should consist of the full context variable name
	// (i.e. `ctx.name`), and the values must be a string. Substitutions defined
	// here take highest precedence over both predefined context variables and
	// evaluated OptionValues.
	//
	// Most users should be using OptionValues to specify custom Job Option values
	// for running the Job instead of using Subsitutions directly.
	//
	// Cannot be updated after creation.
	//
	// +optional
	Substitutions map[string]string `json:"substitutions,omitempty"`

	// Specifies the time to start killing the job. When the time passes this
	// timestamp, the controller will start attempting to kill all tasks.
	//
	// +optional
	KillTimestamp *metav1.Time `json:"killTimestamp,omitempty"`

	// Specifies the maximum lifetime of a Job that is finished. If not set, it will
	// be set to the DefaultTTLSecondsAfterFinished configuration value in the
	// controller.
	//
	// +optional
	TTLSecondsAfterFinished *int64 `json:"ttlSecondsAfterFinished,omitempty"`
}

type JobType string

const (
	// JobTypeAdhoc means that the Job was created on an ad-hoc basis externally.
	JobTypeAdhoc JobType = "Adhoc"

	// JobTypeScheduled means that the Job was created on an automatic schedule.
	JobTypeScheduled JobType = "Scheduled"
)

// StartPolicySpec specifies certain conditions that have to be met before a Job
// can be started.
type StartPolicySpec struct {
	// Specifies the behaviour when there are other concurrent jobs for the
	// JobConfig.
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy"`

	// Specifies the earliest time that the Job can be started after. Can be
	// specified together with other fields.
	//
	// +optional
	StartAfter *metav1.Time `json:"startAfter,omitempty"`
}

type JobTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the job.
	Spec JobTemplateSpec `json:"spec"`
}

type JobTemplateSpec struct {
	// Describes the tasks to be created for the Job.
	Task JobTaskSpec `json:"task"`

	// Specifies maximum number of retry attempts for the Job if the job exceeds its
	// pending timeout or active deadline. Each retry attempt will create a single
	// task at a time. The controller will create up to MaxRetryAttempts+1 tasks for
	// the job, before terminating in RetryLimitExceeded or PendingTimeout. Defaults
	// to 0, which means no retry. Value must be a non-negative integer.
	//
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxRetryAttempts *int32 `json:"maxRetryAttempts,omitempty"`

	// Optional duration in seconds to wait between retries. If left empty or zero,
	// it means no delay (i.e. retry immediately). Value must be a non-negative
	// integer.
	//
	// +kubebuilder:validation:Minimum=0
	// +optional
	RetryDelaySeconds *int64 `json:"retryDelaySeconds,omitempty"`
}

// JobTaskSpec describes a single task in the Job.
type JobTaskSpec struct {
	// Describes how to create tasks as Pods.
	//
	// The following fields support context variable substitution:
	//
	//  - .spec.containers.*.image
	//  - .spec.containers.*.command.*
	//  - .spec.containers.*.args.*
	//  - .spec.containers.*.env.*.value
	Template corev1.PodTemplateSpec `json:"template"`

	// Optional duration in seconds to wait before terminating the task if it is
	// still pending. This field is useful to prevent jobs from being stuck forever
	// if the Job has a deadline to start running by. If not set, it will be set to
	// the DefaultTaskPendingTimeoutSeconds configuration value in the controller.
	//
	// Value must be a positive integer.
	// +optional
	PendingTimeoutSeconds *int64 `json:"pendingTimeoutSeconds,omitempty"`

	// ForbidForceDeletion, if true, means that tasks are not allowed to be
	// force deleted. If the node is unresponsive, it may be possible that the task
	// cannot be killed by normal graceful deletion. The controller may choose to
	// force delete the task, which would ignore the final state of the task since
	// the node is unable to return whether the task is actually still alive.
	//
	// As such, if not set to true, the Forbid ConcurrencyPolicy may in some cases
	// be violated. Setting this to true would prevent this from happening, but the
	// Job may remain in Killing indefinitely until the node recovers.
	//
	// +optional
	ForbidForceDeletion bool `json:"forbidForceDeletion,omitempty"`
}

// JobStatus defines the observed state of a Job.
type JobStatus struct {
	// Phase stores the high-level description of a Job's state.
	Phase JobPhase `json:"phase"`

	// Condition stores details about the Job's current condition.
	Condition JobCondition `json:"condition"`

	// StartTime specifies the time that the Job was started by the controller. If
	// nil, it means that the Job is Queued. Cannot be changed once set.
	//
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CreatedTasks describes how many tasks were created in total for this Job.
	// +optional
	CreatedTasks int64 `json:"createdTasks"`

	// Tasks contains a list of tasks created by the controller. The controller
	// updates this field when it creates a task, which helps to guard against
	// recreating tasks after they were deleted, and also stores necessary task data
	// for reconciliation in case tasks are deleted.
	//
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=atomic
	Tasks []TaskRef `json:"tasks,omitempty"`
}

type JobPhase string

const (
	// JobQueued means that the job is not yet started.
	JobQueued JobPhase = "Queued"

	// JobStarting means that the job is starting up, and no tasks have been created
	// yet.
	JobStarting JobPhase = "Starting"

	// JobPending means that the job is started but not yet running. It could be in
	// the middle of scheduling, container creation, etc.
	JobPending JobPhase = "Pending"

	// JobRunning means that the job has a currently running task.
	JobRunning JobPhase = "Running"

	// JobAdmissionError means that the job could not start due to an admission
	// error that cannot be retried.
	JobAdmissionError JobPhase = "AdmissionError"

	// JobRetryBackoff means that the job is backing off the next retry due to a
	// failed task. The job is currently waiting for its retry delay before creating
	// the next task.
	JobRetryBackoff JobPhase = "RetryBackoff"

	// JobRetrying means that the job had previously failed, and a new task has been
	// created to retry, but it has not yet started running.
	JobRetrying JobPhase = "Retrying"

	// JobSucceeded means that the job was completed successfully.
	JobSucceeded JobPhase = "Succeeded"

	// JobRetryLimitExceeded means that the job's most recent task terminated with a
	// failed result. All retry attempts have been fully exhausted and the job will
	// stop trying to create new tasks.
	JobRetryLimitExceeded JobPhase = "RetryLimitExceeded"

	// JobPendingTimeout means that the job's most recent task did not start running
	// within the maxium pending timeout. All retry attempts have been fully
	// exhausted and the job will stop trying to create new tasks.
	JobPendingTimeout JobPhase = "PendingTimeout"

	// JobDeadlineExceeded means that the job's most recent task had started
	// running, but was running longer than its active deadline. All retry attempts
	// have been fully exhausted and the job will stop trying to create new tasks.
	//
	// Note that the difference between JobPendingTimeout and JobDeadlineExceeded is
	// that the active deadline includes both the pending duration and execution
	// duration (when the container is actually running). If the Job's active
	// deadline is exceeded, and if did not start running within the pending
	// timeout, JobPendingTimeout will be used; if it did start running then
	// JobDeadlineExceeded will be used.
	JobDeadlineExceeded JobPhase = "DeadlineExceeded"

	// JobKilling means that the job and its tasks are in the process of being
	// killed. No more retries will be created.
	JobKilling JobPhase = "Killing"

	// JobKilled means that the job and all its tasks are fully killed via external
	// interference, and tasks are guaranteed to have been stopped. No more tasks
	// will be created even if not all retry attempts are exhausted.
	//
	// It should be noted that if TaskForbidForceDeletion is not true, it may
	// actually be possible that the Node is unresponsive and we had forcefully
	// deleted the task without confirming that the task has been completely killed.
	JobKilled JobPhase = "Killed"

	// JobFinishedUnknown means that the job is finished but for some reason we do
	// not know its result.
	JobFinishedUnknown JobPhase = "FinishedUnknown"
)

// IsTerminal returns true if the Job is considered terminal. A terminal phase
// means that the Job will no longer transition into a non-terminal phase after
// this.
func (p JobPhase) IsTerminal() bool {
	switch p {
	case JobSucceeded,
		JobRetryLimitExceeded,
		JobKilled,
		JobPendingTimeout,
		JobDeadlineExceeded,
		JobAdmissionError,
		JobFinishedUnknown:
		return true

	case JobStarting,
		JobPending,
		JobRunning,
		JobRetryBackoff,
		JobRetrying,
		JobKilling,
		JobQueued:
		fallthrough

	default:
		return false
	}
}

// JobCondition holds a possible condition of a Job.
// Only one of its members may be specified.
// If none of them is specified, the default one is JobConditionQueueing.
type JobCondition struct {
	// Stores the status of the Job's queueing condition. If specified, it means
	// that the Job is currently not started and is queued.
	// +optional
	Queueing *JobConditionQueueing `json:"queueing,omitempty"`

	// Stores the status of the Job's waiting condition. If specified, it means that
	// the Job currently is waiting for a task.
	// +optional
	Waiting *JobConditionWaiting `json:"waiting,omitempty"`

	// Stores the status of the Job's running state. If specified, it means that the
	// Job currently has a running task.
	// +optional
	Running *JobConditionRunning `json:"running,omitempty"`

	// Stores the status of the Job's finished state. If specified, it also means
	// that the Job is terminal.
	// +optional
	Finished *JobConditionFinished `json:"finished,omitempty"`
}

// JobConditionQueueing stores the status of a currently Job in the queue.
type JobConditionQueueing struct {
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Optional descriptive message explaining the condition's last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// JobConditionWaiting stores the status of a currently waiting Job.
type JobConditionWaiting struct {
	// The time at which the latest task was created by the controller, if any.
	// +optional
	CreatedAt *metav1.Time `json:"createTime,omitempty"`

	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Optional descriptive message explaining the condition's last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// JobConditionRunning stores the status of a currently running Job.
type JobConditionRunning struct {
	// The time at which the running task was created by the controller.
	CreatedAt metav1.Time `json:"createTime"`

	// The time at which the running task had started running.
	StartedAt metav1.Time `json:"startTime"`
}

// JobConditionFinished stores the status of the final finished result of a Job.
type JobConditionFinished struct {
	// The time at which the latest task was created by the controller. May be nil
	// if no tasks were ever created.
	// +optional
	CreatedAt *metav1.Time `json:"createTime,omitempty"`

	// The time at which the latest task had started running. May be nil if the task
	// never started running.
	// +optional
	StartedAt *metav1.Time `json:"startTime,omitempty"`

	// The time at which the Job was first marked as finished by the controller.
	FinishedAt metav1.Time `json:"finishTime"`

	// The result of it being finished.
	Result JobResult `json:"result"`

	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Optional descriptive message explaining the condition's last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

type JobResult string

const (
	// JobResultSuccess means that the Job finished successfully.
	JobResultSuccess JobResult = "Success"

	// JobResultTaskFailed means that the Job has failed, and its last task exited
	// with a non-zero code, or encountered some other application-level error.
	JobResultTaskFailed JobResult = "TaskFailed"

	// JobResultPendingTimeout means that the Job has failed to start its last task
	// within the specified pending timeout.
	JobResultPendingTimeout JobResult = "PendingTimeout"

	// JobResultDeadlineExceeded means that the Job has failed to finish its last
	// task within the specified task active deadline.
	JobResultDeadlineExceeded JobResult = "DeadlineExceeded"

	// JobResultAdmissionError means that the Job could not start due to an error
	// from trying to admit creation of tasks.
	JobResultAdmissionError JobResult = "AdmissionError"

	// JobResultKilled means that the Job and its tasks, if any, were successfully
	// killed via KillTimestamp.
	JobResultKilled JobResult = "Killed"

	// JobResultFinalStateUnknown means that the Job's tasks were deleted and its
	// final state is unknown.
	JobResultFinalStateUnknown JobResult = "FinalStateUnknown"
)

var AllFailedJobResults = []JobResult{
	JobResultTaskFailed,
	JobResultPendingTimeout,
	JobResultDeadlineExceeded,
	JobResultAdmissionError,
	JobResultKilled,
}

func (r JobResult) IsFailed() bool {
	for _, state := range AllFailedJobResults {
		if r == state {
			return true
		}
	}
	return false
}

// TaskRef stores information about a Job's owned task.
type TaskRef struct {
	// Name of the task. Assumes to share the same namespace as the Job.
	Name string `json:"name"`

	// Creation time of the task.
	CreationTimestamp metav1.Time `json:"creationTimestamp"`

	// Timestamp that the task transitioned to running. May be zero if the task was
	// never observed as started running.
	//
	// +optional
	RunningTimestamp *metav1.Time `json:"runningTimestamp,omitempty"`

	// Time that the task finished. Will always return a non-zero timestamp if task
	// is finished.
	//
	// +optional
	FinishTimestamp *metav1.Time `json:"finishTimestamp,omitempty"`

	// Status of the task. This field will be reconciled from the relevant task
	// object, may not be always up-to-date. This field will persist the state of
	// tasks beyond the lifetime of the task resources, even if they are deleted.
	Status TaskStatus `json:"status"`

	// DeletedStatus, if set, specifies a placeholder Status of the task after it is
	// reconciled as deleted. If the task is deleted, Status cannot be reconciled
	// from the task any more, and instead uses information stored in DeletedStatus.
	// In other words, this field acts as a tombstone marker, and is only used after
	// the deletion of the task object is complete.
	//
	// While the task is in the process of being deleted (i.e. deletionTimestamp is
	// set but object still exists), Status will still be reconciled from the actual
	// task's status.
	//
	// If the task is already deleted and DeletedStatus is also not set, then the
	// task's state will be marked as TaskDeletedFinalStateUnknown.
	//
	// +optional
	DeletedStatus *TaskStatus `json:"deletedStatus,omitempty"`

	// Node name that the task was bound to. May be empty if task was never
	// scheduled.
	//
	// +optional
	NodeName string `json:"nodeName,omitempty"`

	// States of each container for the task. This field will be reconciled from the
	// relevant task object, and is not guaranteed to be up-to-date. This field will
	// persist the state of tasks beyond the lifetime of the task resources, even if
	// they were deleted.
	ContainerStates []TaskContainerState `json:"containerStates"`
}

// TaskStatus stores the last known status of a Job's task.
type TaskStatus struct {
	// State of the task.
	State TaskState `json:"state"`

	// The execution result derived from this task if it was finished. For
	// simplicity, the values of this field also matches that of the Job's result
	// field.
	//
	// +optional
	Result *JobResult `json:"result,omitempty"`

	// Unique, one-word, CamelCase reason for the task's status.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Descriptive message for the task's status.
	// +optional
	Message string `json:"message,omitempty"`
}

type TaskState string

const (
	// TaskStaging means that the task is created, but no containers are created
	// yet.
	TaskStaging TaskState = "Staging"

	// TaskStarting means that the task is created and container is created, but it
	// has not yet started running.
	TaskStarting TaskState = "Starting"

	// TaskRunning means that the task has started running successfully.
	TaskRunning TaskState = "Running"

	// TaskSuccess means that the task has finished successfully with no errors.
	TaskSuccess TaskState = "Success"

	// TaskFailed means that the task exited with a non-zero code or some other
	// application-level error.
	TaskFailed TaskState = "Failed"

	// TaskPendingTimeout means that the task had failed to start within the
	// specified pending timeout.
	TaskPendingTimeout TaskState = "PendingTimeout"

	// TaskDeadlineExceeded means that the task had failed to terminate within its
	// active deadline and has now been terminated.
	TaskDeadlineExceeded TaskState = "DeadlineExceeded"

	// TaskKilling means that the task is in the process of being killed by external
	// interference.
	TaskKilling TaskState = "Killing"

	// TaskKilled means that the task is successfully killed by external
	// interference.
	TaskKilled TaskState = "Killed"

	// TaskDeletedFinalStateUnknown means that task was deleted and its final status
	// was unknown to the controller. This could happen if the task was force
	// deleted, or the controller lost the status of the task and it was already
	// deleted.
	TaskDeletedFinalStateUnknown TaskState = "DeletedFinalStateUnknown"
)

type TaskContainerState struct {
	// Exit status from the last termination of the container
	ExitCode int32 `json:"exitCode"`

	// Signal from the last termination of the container
	// +optional
	Signal int32 `json:"signal,omitempty"`

	// Unique, one-word, CamelCase reason for the container's status.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message regarding the container's status.
	// +optional
	Message string `json:"message,omitempty"`

	// Container ID of the container. May be empty if the container is not yet
	// created.
	// +optional
	ContainerID string `json:"containerID,omitempty"`
}

// nolint:lll
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=furikojob;furikojobs
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Created Tasks",type=string,JSONPath=`.status.createdTasks`
// +kubebuilder:printcolumn:name="Run Time",type=date,JSONPath=`.status.condition.running.startTime`
// +kubebuilder:printcolumn:name="Finish Time",type=date,JSONPath=`.status.condition.finished.finishTime`
// +kubebuilder:webhook:path=/mutating/jobs.execution.furiko.io,mutating=true,failurePolicy=fail,sideEffects=None,groups=execution.furiko.io,resources=jobs,verbs=create;update,versions=*,name=mutating.webhook.jobs.execution.furiko.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validating/jobs.execution.furiko.io,mutating=false,failurePolicy=fail,sideEffects=None,groups=execution.furiko.io,resources=jobs,verbs=create;update,versions=*,name=validation.webhook.jobs.execution.furiko.io,admissionReviewVersions=v1

// Job is the schema for a single job execution, which may consist of multiple
// tasks.
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobSpec   `json:"spec,omitempty"`
	Status JobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobList contains a list of Job
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Job{}, &JobList{})
}
