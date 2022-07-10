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
	"time"

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
	Template *JobTemplate `json:"template,omitempty"`

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

// JobTemplateSpec specifies how to create a Job with metadata.
type JobTemplateSpec struct {
	// Standard object's metadata that will be added to Job. More info:
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	//
	// +kubebuilder:validation:XPreserveUnknownFields
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the job.
	Spec JobTemplate `json:"spec"`
}

// JobTemplate specifies how to create the Job.
type JobTemplate struct {
	// Defines the template to create a single task in the Job.
	TaskTemplate TaskTemplate `json:"taskTemplate"`

	// Describes how to run multiple tasks in parallel for the Job. If not set, then
	// there will be at most a single task running at any time.
	//
	// +optional
	Parallelism *ParallelismSpec `json:"parallelism,omitempty"`

	// Specifies maximum number of attempts before the Job will terminate in
	// failure. If defined, the controller will wait retryDelaySeconds before
	// creating the next task. Once maxAttempts is reached, the Job terminates in
	// RetryLimitExceeded.
	//
	// If parallelism is also defined, this corresponds to the maximum attempts for
	// each parallel task. That is, if there are 5 parallel task to be run at a
	// time, with maxAttempts of 3, the Job may create up to a maximum of 15 tasks
	// if each has failed.
	//
	// Value must be a positive integer. Defaults to 1.
	//
	// +optional
	MaxAttempts *int64 `json:"maxAttempts,omitempty"`

	// Optional duration in seconds to wait between retries. If left empty or zero,
	// it means no delay (i.e. retry immediately).
	//
	// If parallelism is also defined, the retry delay is from the time of the last
	// failed task with the same index. That is, if there are two parallel tasks -
	// index 0 and index 1 - which failed at t=0 and t=15, with retryDelaySeconds of
	// 30, the controller will only create the next attempts at t=30 and t=45
	// respectively.
	//
	// Value must be a non-negative integer.
	//
	// +optional
	RetryDelaySeconds *int64 `json:"retryDelaySeconds,omitempty"`

	// Optional duration in seconds to wait before terminating the task if it is
	// still pending. This field is useful to prevent jobs from being stuck forever
	// if the Job has a deadline to start running by. If not set, it will be set to
	// the DefaultPendingTimeoutSeconds configuration value in the controller. To
	// disable pending timeout, set this to 0.
	//
	// +optional
	TaskPendingTimeoutSeconds *int64 `json:"taskPendingTimeoutSeconds,omitempty"`

	// Defines whether tasks are allowed to be force deleted or not. If the node is
	// unresponsive, it may be possible that the task cannot be killed by normal
	// graceful deletion. The controller may choose to force delete the task, which
	// would ignore the final state of the task since the node is unable to return
	// whether the task is actually still alive.
	//
	// If not set to true, there may be some cases when there may actually be two
	// concurrently running tasks when even when ConcurrencyPolicyForbid. Setting
	// this to true would prevent this from happening, but the Job may remain stuck
	// indefinitely until the node recovers.
	//
	// +optional
	ForbidTaskForceDeletion bool `json:"forbidTaskForceDeletion,omitempty"`
}

// GetMaxAttempts returns the maximum attempts if specified in the template, otherwise defaults to 1.
func (j *Job) GetMaxAttempts() int64 {
	var maxAttempts int64 = 1
	if template := j.Spec.Template; template != nil {
		if j.Spec.Template.MaxAttempts != nil {
			maxAttempts = *j.Spec.Template.MaxAttempts
		}
	}
	return maxAttempts
}

// GetRetryDelay return the retry delay if specified in the template, otherwise defaults to 0.
func (j *Job) GetRetryDelay() time.Duration {
	var delay time.Duration
	if template := j.Spec.Template; template != nil {
		if j.Spec.Template.RetryDelaySeconds != nil {
			delay = time.Duration(*template.RetryDelaySeconds) * time.Second
		}
	}
	return delay
}

// ParallelismSpec specifies how to run multiple tasks in parallel in a Job.
type ParallelismSpec struct {
	// Specifies an exact number of tasks to be run in parallel. The index number
	// can be retrieved via the "${task.index_num}" context variable.
	//
	// +optional
	WithCount *int64 `json:"withCount,omitempty"`

	// Specifies a list of keys corresponding to each task that will be run in
	// parallel. The index key can be retrieved via the "${task.index_key}" context
	// variable.
	//
	// +listType=atomic
	// +optional
	WithKeys []string `json:"withKeys,omitempty"`

	// Specifies a matrix of key-value pairs, with each key mapped to a list of
	// possible values, such that tasks will be started for each combination of
	// key-value pairs. The matrix values can be retrieved via context variables in
	// the following format: "${task.index_matrix.<key>}".
	//
	// +mapType=atomic
	// +optional
	WithMatrix map[string][]string `json:"withMatrix,omitempty"`

	// Defines when the Job will complete when there are multiple tasks running in
	// parallel. For example, if using the AllSuccessful strategy, the Job will only
	// terminate once all parallel tasks have terminated successfully, or once any
	// task has exhausted its maxAttempts limit.
	//
	// +optional
	CompletionStrategy ParallelCompletionStrategy `json:"completionStrategy,omitempty"`
}

// GetCompletionStrategy returns the completion strategy, otherwise returns a default.
func (in *ParallelismSpec) GetCompletionStrategy() ParallelCompletionStrategy {
	if strategy := in.CompletionStrategy; strategy != "" {
		return strategy
	}
	return AllSuccessful
}

// ParallelCompletionStrategy defines the condition when a Job is completed when
// there are multiple tasks running in parallel.
type ParallelCompletionStrategy string

const (
	// AllSuccessful means that the Job will only stop once all parallel tasks have
	// completed successfully, or when any task index has exhausted all its retries
	// and immediately terminate all remaining tasks.
	AllSuccessful ParallelCompletionStrategy = "AllSuccessful"

	// AnySuccessful means that the Job will stop once any parallel task has
	// completed successfully and immediately terminate all remaining tasks.
	AnySuccessful ParallelCompletionStrategy = "AnySuccessful"
)

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
	CreatedTasks int64 `json:"createdTasks"`

	// RunningTasks describes how many tasks are currently running for this Job.
	RunningTasks int64 `json:"runningTasks"`

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

	// The current status for parallel execution of the job.
	// May not be set if the job is not a parallel job.
	//
	// +optional
	ParallelStatus *ParallelStatus `json:"parallelStatus,omitempty"`
}

type JobPhase string

const (
	// JobQueued means that the job is not yet started.
	JobQueued JobPhase = "Queued"

	// JobStarting means that the job is starting up, and no tasks have been created
	// yet.
	JobStarting JobPhase = "Starting"

	// JobAdmissionError means that the job could not start due to an admission
	// error that cannot be retried.
	JobAdmissionError JobPhase = "AdmissionError"

	// JobPending means that the job is started but not all tasks are running yet.
	JobPending JobPhase = "Pending"

	// JobRunning means that all tasks have started running, and it is not yet
	// finished.
	JobRunning JobPhase = "Running"

	// JobTerminating means that the job is completed, but not all tasks are
	// terminated yet.
	JobTerminating JobPhase = "Terminating"

	// JobRetryBackoff means that the job is backing off the next retry due to a
	// failed task. The job is currently waiting for its retry delay before creating
	// the next task.
	JobRetryBackoff JobPhase = "RetryBackoff"

	// JobRetrying means that the job had previously failed, and a new task has been
	// created to retry, but it has not yet started running.
	JobRetrying JobPhase = "Retrying"

	// JobSucceeded means that the job was completed successfully.
	JobSucceeded JobPhase = "Succeeded"

	// JobFailed means that the job did not complete successfully.
	JobFailed JobPhase = "Failed"

	// JobKilling means that the job and its tasks are in the process of being
	// killed and no more tasks will be created.
	JobKilling JobPhase = "Killing"

	// JobKilled means that the job and all its tasks are fully killed via external
	// interference, and tasks are guaranteed to have been stopped. No more tasks
	// will be created even if not all retry attempts are exhausted.
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
		JobFailed,
		JobKilled,
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
	//
	// +optional
	Queueing *JobConditionQueueing `json:"queueing,omitempty"`

	// Stores the status of the Job's waiting condition. If specified, it means that
	// the Job currently is waiting for at least one task to be created and start
	// running.
	//
	// +optional
	Waiting *JobConditionWaiting `json:"waiting,omitempty"`

	// Stores the status of the Job's running state. If specified, it means that all
	// tasks in the Job have started running. If the Job is already complete, this
	// status may be set of not all tasks are terminated.
	//
	// +optional
	Running *JobConditionRunning `json:"running,omitempty"`

	// Stores the status of the Job's finished state. If specified, it means that
	// all tasks in the Job have terminated.
	//
	// +optional
	Finished *JobConditionFinished `json:"finished,omitempty"`
}

// JobConditionQueueing stores the status of a Job currently in the queue.
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
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Optional descriptive message explaining the condition's last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// JobConditionRunning stores the status of a currently running Job.
type JobConditionRunning struct {
	// The timestamp for the latest task that was created by the controller.
	LatestCreationTimestamp metav1.Time `json:"latestTaskCreationTimestamp"`

	// The time at which the latest task had started running.
	LatestRunningTimestamp metav1.Time `json:"latestRunningTimestamp"`

	// Number of tasks waiting to be terminated.
	TerminatingTasks int64 `json:"terminatingTasks,omitempty"`
}

// JobConditionFinished stores the status of the final finished result of a Job.
type JobConditionFinished struct {
	// The time at which the latest task was created by the controller. May be nil
	// if no tasks were ever created.
	//
	// +optional
	LatestCreationTimestamp *metav1.Time `json:"latestCreationTimestamp,omitempty"`

	// The time at which the latest task had started running. May be nil if no tasks
	// had started running.
	//
	// +optional
	LatestRunningTimestamp *metav1.Time `json:"latestRunningTimestamp,omitempty"`

	// The time at which the Job was first marked as finished by the controller.
	FinishTimestamp metav1.Time `json:"finishTime"`

	// The result of it being finished.
	Result JobResult `json:"result"`

	// Unique, one-word, CamelCase reason for the condition's last transition.
	//
	// +optional
	Reason string `json:"reason,omitempty"`

	// Optional descriptive message explaining the condition's last transition.
	//
	// +optional
	Message string `json:"message,omitempty"`
}

type JobResult string

const (
	// JobResultSuccess means that the Job finished successfully.
	JobResultSuccess JobResult = "Success"

	// JobResultFailed means that the Job has failed and will no longer retry.
	JobResultFailed JobResult = "Failed"

	// JobResultAdmissionError means that the Job could not start due to an error
	// from trying to admit creation of tasks.
	JobResultAdmissionError JobResult = "AdmissionError"

	// JobResultKilled means that the Job and all of its tasks have been killed.
	JobResultKilled JobResult = "Killed"

	// JobResultFinalStateUnknown means that the final state of the job is unknown.
	JobResultFinalStateUnknown JobResult = "FinalStateUnknown"
)

var AllFailedJobResults = []JobResult{
	JobResultFailed,
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

	// The retry index of the task, starting from 0 up to maxAttempts - 1.
	RetryIndex int64 `json:"retryIndex"`

	// If the Job is a parallel job, then contains the parallel index of the task.
	//
	// +optional
	ParallelIndex *ParallelIndex `json:"parallelIndex"`

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
	//
	// +optional
	ContainerStates []TaskContainerState `json:"containerStates,omitempty"`
}

// TaskStatus stores the last known status of a Job's task.
type TaskStatus struct {
	// State of the task.
	State TaskState `json:"state"`

	// If the state is Terminated, the result of the task.
	// +optional
	Result TaskResult `json:"result,omitempty"`

	// Unique, one-word, CamelCase reason for the task's status.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Descriptive message for the task's status.
	// +optional
	Message string `json:"message,omitempty"`
}

type TaskState string

const (
	// TaskStarting means that the task has not yet started running.
	TaskStarting TaskState = "Starting"

	// TaskRunning means that the task has started running successfully.
	TaskRunning TaskState = "Running"

	// TaskKilling means that the task is being killed externally but has not yet terminated.
	TaskKilling TaskState = "Killing"

	// TaskTerminated means that the task is terminated.
	TaskTerminated TaskState = "Terminated"

	// TaskDeletedFinalStateUnknown means that task was deleted and its final status
	// was unknown to the controller. This could happen if the task was force
	// deleted, or the controller lost the status of the task and it was already
	// deleted.
	TaskDeletedFinalStateUnknown TaskState = "DeletedFinalStateUnknown"
)

// TaskResult contains the result of a Task.
type TaskResult string

const (
	// TaskSucceeded means that the task finished successfully with no errors.
	TaskSucceeded TaskResult = "Succeeded"

	// TaskFailed means that the task exited with a non-zero code or some other
	// application-level error.
	TaskFailed TaskResult = "Failed"

	// TaskKilled means that the task is killed externally and now terminated.
	TaskKilled TaskResult = "Killed"
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

// ParallelIndex specifies the index for a single parallel task.
type ParallelIndex struct {
	// If withCount is used for parallelism, contains the index number of the job
	// numbered from 0 to N-1.
	//
	// +optional
	IndexNumber *int64 `json:"indexNumber,omitempty"`

	// If withKeys is used for parallelism, contains the index key of the job as a
	// string.
	//
	// +optional
	IndexKey string `json:"indexKey,omitempty"`

	// If withMatrix is used for parallelism, contains key-value pairs of the job as
	// strings.
	//
	// +mapType=atomic
	// +optional
	MatrixValues map[string]string `json:"matrixValues,omitempty"`
}

// ParallelStatus stores the status of parallel indexes for a Job.
type ParallelStatus struct {
	ParallelStatusSummary `json:",inline"`

	// The status for each parallel index. The size of the list should be exactly
	// equal to the total parallelism factor, even if no tasks are created yet.
	Indexes []ParallelIndexStatus `json:"indexes"`
}

// ParallelIndexStatus stores the status for a single ParallelIndex in the Job.
// There should be at most one task running at a time for a single parallel
// index in the Job.
type ParallelIndexStatus struct {
	// The parallel index.
	Index ParallelIndex `json:"index"`

	// Hash of the index.
	Hash string `json:"hash"`

	// Total number of tasks created for this parallel index.
	CreatedTasks int64 `json:"createdTasks"`

	// Overall state of the parallel index.
	State IndexState `json:"state"`

	// Result of executing tasks for this parallel index if it is already terminated.
	// +optional
	Result TaskResult `json:"result,omitempty"`
}

type IndexState string

const (
	// IndexNotCreated means that no tasks have been created for the index.
	IndexNotCreated IndexState = "NotCreated"

	// IndexRetryBackoff means that the index is currently in retry backoff.
	IndexRetryBackoff IndexState = "RetryBackoff"

	// IndexStarting means that the current task in the index has not yet started running.
	IndexStarting IndexState = "Starting"

	// IndexRunning means that the current task in the index is currently running.
	IndexRunning IndexState = "Running"

	// IndexTerminated means that all tasks in the index is terminated.
	IndexTerminated IndexState = "Terminated"
)

// ParallelStatusSummary stores the summary status of parallel indexes for a Job.
type ParallelStatusSummary struct {
	// If true, the job is complete and currently in the process of waiting for all
	// remaining tasks to be terminated.
	Complete bool `json:"complete"`

	// If complete, contains whether the job is successful according to the
	// ParallelCompletionStrategy.
	//
	// +optional
	Successful *bool `json:"successful,omitempty"`
}

// ParallelStatusCounters stores counts of parallel indexes for a Job.
type ParallelStatusCounters struct {
	// Total number of task parallel indexes that have created tasks.
	Created int64 `json:"created"`

	// Total number of task parallel indexes that are currently starting.
	Starting int64 `json:"starting"`

	// Total number of task parallel indexes that are currently running.
	Running int64 `json:"running"`

	// Total number of task parallel indexes that are currently in retry backoff.
	RetryBackoff int64 `json:"retryBackoff"`

	// Total number of task parallel indexes where all tasks are completely
	// terminated, including when no tasks are created.
	Terminated int64 `json:"terminated"`

	// Total number of task parallel indexes that are successful.
	Succeeded int64 `json:"succeeded"`

	// Total number of task parallel indexes that failed (including all retries).
	Failed int64 `json:"failed"`
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
// +kubebuilder:printcolumn:name="Running Tasks",type=string,JSONPath=`.status.runningTasks`
// +kubebuilder:printcolumn:name="Run Time",type=date,JSONPath=`.status.condition.running.latestRunningTimestamp`
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
