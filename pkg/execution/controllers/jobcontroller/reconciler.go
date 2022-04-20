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

package jobcontroller

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	rerrors "github.com/furiko-io/furiko/pkg/errors"
	jobtasks "github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/execution/variablecontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/utils/k8sutils"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	timeutil "github.com/furiko-io/furiko/pkg/utils/time"
)

type Reconciler struct {
	*Context
	client      *ExecutionControl
	concurrency *configv1alpha1.Concurrency
}

func NewReconciler(ctrlContext *Context, concurrency *configv1alpha1.Concurrency) *Reconciler {
	worker := &Reconciler{
		Context:     ctrlContext,
		concurrency: concurrency,
	}
	worker.client = NewExecutionControl(ctrlContext.Clientsets().Furiko().ExecutionV1alpha1(), worker.Name())
	return worker
}

func (w *Reconciler) Name() string {
	return fmt.Sprintf("%v.Reconciler", controllerName)
}

func (w *Reconciler) Concurrency() int {
	return controllerutil.GetConcurrencyOrDefaultCPUFactor(w.concurrency, 4)
}

func (w *Reconciler) MaxRequeues() int {
	return -1
}

func (w *Reconciler) SyncOne(ctx context.Context, namespace, name string, _ int) error {
	var err error

	cfg, err := w.Configs().Jobs()
	if err != nil {
		return errors.Wrapf(err, "cannot load controller configuration")
	}

	trace := utiltrace.New(
		"job_sync",
		utiltrace.Field{Key: "namespace", Value: namespace},
		utiltrace.Field{Key: "name", Value: name},
	)
	defer trace.LogIfLong(500 * time.Millisecond)
	klog.V(2).InfoS("jobcontroller: syncing job",
		"worker", w.Name(),
		"namespace", namespace,
		"name", name,
	)

	rj, err := w.jobInformer.Lister().Jobs(namespace).Get(name)
	if kerrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "cannot get job")
	}
	trace.Step("Lookup job from cache done")

	// Perform sync. We assume that the Job is never modified in-place by any downstream function.
	newRj, syncErr := w.sync(ctx, rj, cfg, trace)

	// Should not enter here, otherwise it means that there is a bug in the reconciler.
	if newRj == nil {
		klog.Warningf("jobcontroller: nil job was returned after sync, worker=%v namespace=%v name=%v", w.Name(),
			namespace, name)
		return syncErr
	}

	// Update Job and JobStatus.
	if _, err := w.client.UpdateJob(ctx, rj, newRj); err != nil {
		return errors.Wrapf(err, "cannot update job")
	}

	// Update the JobStatus if different.
	if _, err := w.client.UpdateJobStatus(ctx, rj, newRj); err != nil {
		return errors.Wrapf(err, "cannot update job")
	}

	return syncErr
}

// sync performs the main logic for the controller.
// IMPORTANT: This method MUST returns the latest Job we want to update, even if an error was met midway.
func (w *Reconciler) sync(
	ctx context.Context, rj *execution.Job, cfg *configv1alpha1.JobExecutionConfig, trace *utiltrace.Trace,
) (*execution.Job, error) {
	// Main logic: Perform task creation/adoption and reconciliation. If Job is not
	// started or is being deleted, this is a no-op.
	if job.IsStarted(rj) && !isDeleted(rj) {
		updatedRj, err := w.syncJobTasks(ctx, rj, cfg, trace)
		if err != nil {
			return rj, errors.Wrapf(err, "could not sync job tasks")
		}
		rj = updatedRj
		trace.Step("Sync job tasks")
	}

	// Compute JobStatus.
	// TODO(irvinlim): This performs duplicate work from syncJobTasks but is
	//  unfortunately necessary to handle non-started Jobs.
	rj = w.syncJobStatusFromTaskRefs(rj)

	// Clean up Job if it is finished and beyond its TTL.
	if err := w.handleTTLAfterFinished(ctx, rj, cfg); err != nil {
		return rj, errors.Wrapf(err, "could not handle TTLAfterFinished")
	}
	trace.Step("Handle TTLAfterFinished done")

	// Finalize Job if deleting.
	updatedRj, err := w.handleFinishFinalizer(ctx, rj)
	if err != nil {
		return rj, errors.Wrapf(err, "could not finalize %v", executiongroup.DeleteDependentsFinalizer)
	}
	rj = updatedRj
	trace.Step("Handle finalizer done")

	return rj, nil
}

// syncJobTasks performs the main reconciliation logic for managing tasks and reconciling the status with the tasks.
func (w *Reconciler) syncJobTasks(
	ctx context.Context, rj *execution.Job, cfg *configv1alpha1.JobExecutionConfig, trace *utiltrace.Trace,
) (*execution.Job, error) {
	taskMgr, err := w.tasks.ForJob(rj)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create task manager")
	}

	maxAllowedTasks := job.GetMaxAllowedTasks(rj)

	// Get all tasks for job from cache.
	tasks, err := taskMgr.Lister().List()
	if err != nil {
		return rj, errors.Wrapf(err, "could not list tasks")
	}
	trace.Step("List tasks from cache done")

	// Skip adoption step if there is an active task.
	// We assume there is only 1 task active at a time.
	if !job.ContainsActiveTask(tasks) {
		// Get the maximum retry index out of all tasks.
		// This may not contain all tasks created by the controller, only the non-deleted ones in the cache.
		maxRetryIndex := job.MaxTaskRetryIndex(tasks)

		// Try to adopt task(s) until exceed retries.
		// We assume this should not take too long because the cache should be almost up-to-date with apiserver.
		for i := maxRetryIndex + 1; i <= maxAllowedTasks; i++ {
			// Attempt to fetch task with next retry index from server
			task, err := taskMgr.Client().Index(ctx, i)
			if kerrors.IsNotFound(err) {
				break
			} else if err != nil {
				return rj, errors.Wrapf(err, "could not fetch task index %v", i)
			}

			// Found a task, add to our list.
			tasks = append(tasks, task)
		}
		trace.Step("Look for retries to adopt done")
	}

	// After adopting tasks, ensure that CreatedTask status is up-to-date for use later.
	rj = w.updateTaskRefStatus(rj, tasks)
	trace.Step("Update status from adopted tasks done")

	// Determine if we need to create new task based on status.
	newRj, tasks, err := w.syncCreateNewTask(ctx, rj, tasks)
	if err != nil {
		return rj, err
	}
	rj = newRj
	trace.Step("Create new task done")

	// Check if any tasks exceed pending timeout.
	if err := w.handlePendingTasks(ctx, rj, tasks, cfg); err != nil {
		return rj, errors.Wrapf(err, "could not reap pending tasks")
	}
	trace.Step("Reap overdue pending tasks done")

	// Handle propagation of kill timestamp to all unfinished tasks
	if err := w.handleKillJob(ctx, rj, tasks); err != nil {
		return rj, errors.Wrapf(err, "could not kill job")
	}
	trace.Step("Set kill timestamp on tasks done")

	// Use deletion of tasks when previous kill is ineffective.
	newRj, err = w.handleDeleteKillingTasks(ctx, rj, tasks, cfg)
	if err != nil {
		return rj, errors.Wrapf(err, "could not delete killing tasks")
	}
	rj = newRj
	trace.Step("Delete unkillable tasks done")

	// Use force deletion of tasks when previous delete is ineffective.
	newRj, err = w.handleForceDeleteKillingTasks(ctx, rj, tasks, cfg)
	if err != nil {
		return rj, errors.Wrapf(err, "could not force delete killing tasks")
	}
	rj = newRj
	trace.Step("Force delete undeletable tasks done")

	// Final update of task refs.
	rj = w.updateTaskRefStatus(rj, tasks)
	trace.Step("Final update status for tasks done")

	return rj, nil
}

// updateTaskRefStatus will update the CreatedTask fields in the Job's status from a list of tasks.
// We will always update tasks into Job before computing the rest of the JobStatus.
func (w *Reconciler) updateTaskRefStatus(rj *execution.Job, tasks []jobtasks.Task) *execution.Job {
	// Generate new TaskRefs in Job status.
	updatedRj := job.UpdateJobTaskRefs(rj, tasks)

	// Update Job status with new TaskRefs.
	return w.syncJobStatusFromTaskRefs(updatedRj)
}

// syncJobStatusFromTaskRefs will sync the rest of the JobStatus from CreatedTaskRefs.
// The exact flow of data is: PodStatus -> CreatedTaskRefs -> Condition -> Phase.
func (w *Reconciler) syncJobStatusFromTaskRefs(rj *execution.Job) *execution.Job {
	newRj := UpdateJobStatusFromTaskRefs(rj)

	// Handle job that is newly running.
	if rj.Status.Condition.Running == nil && newRj.Status.Condition.Running != nil {
		w.recorder.Eventf(newRj, corev1.EventTypeNormal, "Running",
			"Job started running")
	}

	// Handle job that is newly finished.
	if rj.Status.Condition.Finished == nil && newRj.Status.Condition.Finished != nil {
		result := newRj.Status.Condition.Finished.Result
		if result.IsFailed() {
			w.recorder.Eventf(newRj, corev1.EventTypeWarning, "Failed",
				"Job failed with result: %v", result)
		} else {
			w.recorder.Eventf(newRj, corev1.EventTypeNormal, "Finished",
				"Job finished with result: %v", result)
		}
	}

	// Enqueue work to delete finished Job after TTL.
	if newRj.Status.Condition.Finished != nil && !isDeleted(newRj) {
		if newRj.Spec.TTLSecondsAfterFinished != nil {
			timeout := time.Duration(*newRj.Spec.TTLSecondsAfterFinished) * time.Second
			duration := time.Until(newRj.Status.Condition.Finished.FinishedAt.Add(timeout))
			w.enqueueAfter(rj, "ttl_seconds_after_finished", duration)
		}
	}

	return newRj
}

// UpdateJobStatusFromTaskRefs returns a new Job after updating JobStatus from TaskRefs.
func UpdateJobStatusFromTaskRefs(rj *execution.Job) *execution.Job {
	newRj := rj.DeepCopy()

	// Compute consolidated condition for Job.
	newRj.Status.Condition = job.GetCondition(rj)

	// If job is being deleted and is not properly finished (e.g. being deleted midway),
	// we use Killed as the final result for the job.
	// This ensures that the final status before the job is finalized (during deletion) is terminal.
	if newRj.DeletionTimestamp != nil && newRj.Status.Condition.Finished == nil {
		createdAt := *ktime.Now()
		var startedAt *metav1.Time
		if condition := newRj.Status.Condition.Waiting; condition != nil && condition.CreatedAt != nil {
			createdAt = *condition.CreatedAt
		} else if condition := newRj.Status.Condition.Running; condition != nil {
			createdAt = condition.CreatedAt
			startedAt = &condition.StartedAt
		}

		newRj.Status.Condition = execution.JobCondition{
			Finished: &execution.JobConditionFinished{
				CreatedAt:  &createdAt,
				StartedAt:  startedAt,
				FinishedAt: createdAt,
				Result:     execution.JobResultKilled,
			},
		}
	}

	// Set phase based on computed status so far.
	newRj.Status.Phase = job.GetPhase(newRj)

	return newRj
}

// syncCreateNewTask will determine if a new task needs to be created, and if
// so, create it and update the list of tasks with the newly created task. At
// this point, the entire JobStatus should be up-to-date. We use
// CreatedTask fields in the Status to authoritatively determine how many tasks
// have been created, but use the tasks list to get the list of non-deleted
// tasks.
func (w *Reconciler) syncCreateNewTask(
	ctx context.Context, rj *execution.Job, tasks []jobtasks.Task,
) (*execution.Job, []jobtasks.Task, error) {
	now := ktime.Now().Time

	// Not allowed to create new task.
	if !job.AllowedToCreateNewTask(rj) {
		return rj, tasks, nil
	}

	// Delay creating new task if RetryDelay is set.
	earliestCreateTime, err := job.GetNextAllowedRetry(rj)
	if err != nil {
		return rj, tasks, errors.Wrapf(err, "cannot get next allowed retry")
	} else if earliestCreateTime.After(now) {
		w.enqueueAfter(rj, "retry_delay_create_task", time.Until(earliestCreateTime))
		return rj, tasks, nil
	}

	// Create new task.
	task, err := w.createTask(ctx, rj)
	if err != nil {
		// Handle this as a normal error.
		rerr := rerrors.Error(nil)
		if !errors.As(err, &rerr) || !rerrors.IsAdmissionRefused(err) {
			return rj, tasks, errors.Wrapf(err, "could not create task")
		}

		// Cannot create task due to unretryable error.
		// Give up trying to sync the task further, so we conclude it cannot be created.
		newRj := rj.DeepCopy()
		job.MarkAdmissionError(newRj, rerr.Error())
		rj = newRj

		// Publish event.
		klog.ErrorS(rerr, "jobcontroller: worker cannot create task",
			"worker", w.Name(),
			"namespace", rj.GetNamespace(),
			"name", rj.GetName(),
		)
		w.recorder.Eventf(rj, corev1.EventTypeWarning, "AdmissionError",
			"Cannot create task: %v", rerr)
	} else {
		// Record event.
		klog.InfoS("jobcontroller: worker created task",
			"worker", w.Name(),
			"namespace", rj.GetNamespace(),
			"name", rj.GetName(),
			"task", task.GetName(),
		)
		w.recorder.Eventf(rj, corev1.EventTypeNormal, "Created",
			"Created task of kind %v: %v", task.GetKind(), task.GetName())

		// Add to our list of tasks.
		tasks = append(tasks, task)
	}

	// Sync Job's status with the new list of tasks before moving on.
	updatedRj := w.updateTaskRefStatus(rj, tasks)

	return updatedRj, tasks, nil
}

// createTask will create a new task for the Job.
// May return AdmissionError if it cannot be created due to an unretryable or irrecoverable error.
func (w *Reconciler) createTask(ctx context.Context, rj *execution.Job) (jobtasks.Task, error) {
	// Substitute job context variables.
	newRj := rj.DeepCopy()
	newRj.Spec.Template.Task.Template = variablecontext.SubstitutePodTemplateSpecForJob(rj)

	// Get task manager after substituting variables in Job.
	taskMgr, err := w.tasks.ForJob(newRj)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get task manager")
	}

	// Create new task.
	task, err := taskMgr.Client().CreateIndex(ctx, rj.Status.CreatedTasks+1)
	if err != nil {
		return nil, err
	}

	// Observe latency for initial task creation.
	ObserveFirstTaskCreation(rj, task)

	return task, nil
}

// handlePendingTasks looks for pending tasks that have exceeded their pending timeout, and subsequently
// kill those tasks.
func (w *Reconciler) handlePendingTasks(ctx context.Context, rj *execution.Job, tasks []jobtasks.Task,
	cfg *configv1alpha1.JobExecutionConfig) error {
	now := ktime.Now().Time

	pendingTimeout := job.GetPendingTimeout(rj, cfg)

	// Pending timeout is disabled.
	if pendingTimeout <= 0 {
		return nil
	}

	// Find tasks that need to be killed.
	needKill := make([]jobtasks.Task, 0, len(tasks))
	for _, task := range tasks {
		ref := task.GetTaskRef()

		// Skip if task already finished.
		if ts := ref.FinishTimestamp; !ts.IsZero() {
			continue
		}

		// Skip if task already started running.
		if ts := ref.RunningTimestamp; !ts.IsZero() {
			continue
		}

		// Skip if task is not yet overdue.
		if deadline := ref.CreationTimestamp.Add(pendingTimeout); deadline.After(now) {
			w.enqueueAfter(rj, "task_pending_timeout", time.Until(deadline))
			continue
		}

		// Found task to kill.
		needKill = append(needKill, task)

		klog.InfoS("jobcontroller: reaping overdue pending task",
			"worker", w.Name(),
			"namespace", rj.GetNamespace(),
			"name", rj.GetName(),
			"task", ref.Name,
		)
	}

	if len(needKill) == 0 {
		return nil
	}

	// Mark tasks as killed due to pending timeout.
	if err := job.ConcurrentTasks(needKill, func(task jobtasks.Task) error {
		if task.GetKilledFromPendingTimeoutMarker() {
			return nil
		}
		if err := task.SetKilledFromPendingTimeoutMarker(ctx); err != nil {
			return err
		}

		klog.InfoS("jobcontroller: set task as killed from pending timeout",
			"worker", w.Name(),
			"namespace", rj.GetNamespace(),
			"name", rj.GetName(),
			"task", task.GetName(),
		)

		return nil
	}); err != nil {
		return errors.Wrapf(err, "could not set task(s) as killed from pending timeout")
	}

	// Set kill timestamp on tasks.
	if err := w.setTasksKillTimestamp(ctx, rj, needKill, *ktime.Now()); err != nil {
		return err
	}

	return nil
}

// handleKillJob updates kill timestamp of all tasks if spec.killTimestamp is set.
func (w *Reconciler) handleKillJob(ctx context.Context, rj *execution.Job, tasks []jobtasks.Task) error {
	// Skip if not killing.
	if rj.Spec.KillTimestamp == nil || ktime.Now().Before(rj.Spec.KillTimestamp) {
		return nil
	}

	// Find all tasks that need update.
	needUpdate := make([]jobtasks.Task, 0, len(tasks))
	for _, task := range tasks {
		// Skip finished tasks
		if job.IsTaskFinished(task) {
			continue
		}

		// Skip if kill timestamp exists, and existing one is earlier than the killTimestamp we requested.
		if ktime.IsTimeSetAndEarlierThanOrEqualTo(task.GetKillTimestamp(), rj.Spec.KillTimestamp.Time) {
			continue
		}

		needUpdate = append(needUpdate, task)
	}

	if len(needUpdate) == 0 {
		return nil
	}

	// Set the kill timestamp for tasks.
	if err := w.setTasksKillTimestamp(ctx, rj, needUpdate, *rj.Spec.KillTimestamp); err != nil {
		return err
	}

	return nil
}

// handleDeleteKillingTasks uses deletion to kill tasks if prior efforts to set kill timestamp on tasks are ineffective.
func (w *Reconciler) handleDeleteKillingTasks(
	ctx context.Context, rj *execution.Job, tasks []jobtasks.Task, cfg *configv1alpha1.JobExecutionConfig,
) (*execution.Job, error) {
	timeout := job.GetDeleteKillingTimeout(cfg)
	needDelete := make([]jobtasks.Task, 0, len(tasks))
	needDeleteMap := make(map[string]jobtasks.Task)

	for _, task := range tasks {
		// Skip finished tasks.
		if job.IsTaskFinished(task) {
			continue
		}

		// Skip if task is not scheduled to be killed.
		killTS := task.GetKillTimestamp()
		if killTS == nil || killTS.IsZero() {
			continue
		}

		// Don't need to delete if deletionTimestamp already set and earlier.
		if ts := task.GetDeletionTimestamp(); ktime.IsTimeSetAndEarlier(ts) {
			continue
		}

		// If task is not killable with kill timestamp, or is still alive beyond kill timestamp + timeout,
		// use deletion to kill the task instead.
		if !killTS.Add(timeout).After(ktime.Now().Time) || task.RequiresKillWithDeletion() {
			klog.InfoS("jobcontroller: worker deleting killing task",
				"worker", w.Name(),
				"namespace", rj.GetNamespace(),
				"name", rj.GetName(),
				"task", task.GetName(),
				"killTimestamp", killTS,
			)

			needDelete = append(needDelete, task)
			needDeleteMap[task.GetName()] = task
		} else {
			// Otherwise, enqueue sync after timeout.
			w.enqueueAfter(rj, "delete_killing_task", time.Until(killTS.Add(timeout)))
		}
	}

	if len(needDelete) == 0 {
		return rj, nil
	}

	newRj := rj.DeepCopy()

	// Update DeletedStatus, will be updated to TaskRef's status once actually deleted.
	newRefs := make([]execution.TaskRef, 0, len(newRj.Status.Tasks))
	for _, taskRef := range newRj.Status.Tasks {
		newRef := taskRef.DeepCopy()
		if task, ok := needDeleteMap[taskRef.Name]; ok {
			// Assume that task is being killed.
			newRef.DeletedStatus = &execution.TaskStatus{
				State:   execution.TaskKilled,
				Result:  job.GetResultPtr(execution.JobResultKilled),
				Reason:  "Deleted",
				Message: "Task was killed via deletion",
			}

			// Set unschedulable reason if was marked to be killed by pending timeout.
			if killedFromPending := task.GetKilledFromPendingTimeoutMarker(); killedFromPending {
				newRef.DeletedStatus.Result = job.GetResultPtr(execution.JobResultPendingTimeout)
			}
		}

		newRefs = append(newRefs, *newRef)
	}
	newRj.Status.Tasks = newRefs

	// Compute new task status.
	newRj = job.UpdateJobTaskRefs(newRj, tasks)

	// Delete tasks.
	if err := w.deleteTasks(ctx, newRj, needDelete, false); err != nil {
		return newRj, err
	}

	return newRj, nil
}

// handleForceDeleteKillingTasks will perform non-graceful deletion of tasks if it cannot be deleted gracefully.
// For example, when kubelet is not running, normal deletion will not be effective.
func (w *Reconciler) handleForceDeleteKillingTasks(
	ctx context.Context, rj *execution.Job, tasks []jobtasks.Task, cfg *configv1alpha1.JobExecutionConfig,
) (*execution.Job, error) {
	timeout := job.GetForceDeleteKillingTimeout(cfg)

	// Force deletion is disabled.
	if timeout <= 0 {
		return rj, nil
	}

	needDelete := make([]jobtasks.Task, 0, len(tasks))
	needDeleteMap := make(map[string]jobtasks.Task)

	// This Job's tasks cannot be force deleted.
	// Do not continue with the rest of the routine.
	if rj.Spec.Template.Task.ForbidForceDeletion {
		// TODO(irvinlim): Check if we need special handling here.
		return rj, nil
	}

	for _, task := range tasks {
		deletionTS := task.GetDeletionTimestamp()
		if deletionTS.IsZero() {
			continue
		}

		if !deletionTS.Add(timeout).After(ktime.Now().Time) {
			klog.InfoS("jobcontroller: worker force deleting killing task",
				"worker", w.Name(),
				"namespace", rj.GetNamespace(),
				"name", rj.GetName(),
				"task", task.GetName(),
				"deletionTimestamp", deletionTS,
			)

			needDelete = append(needDelete, task)
			needDeleteMap[task.GetName()] = task
		} else {
			// Otherwise, enqueue sync after timeout.
			w.enqueueAfter(rj, "force_delete_killing_task", time.Until(deletionTS.Add(timeout)))
		}
	}

	if len(needDelete) == 0 {
		return rj, nil
	}

	// Before deleting, update task message to mention that it was force deleted.
	// We can do this by storing in DeletedStatus.
	newRj := rj.DeepCopy()

	// Update DeletedStatus, will be updated to TaskRef's status once actually deleted.
	newRefs := make([]execution.TaskRef, 0, len(newRj.Status.Tasks))
	for _, taskRef := range newRj.Status.Tasks {
		newRef := taskRef.DeepCopy()
		if task, ok := needDeleteMap[taskRef.Name]; ok {
			newRef.DeletedStatus = &execution.TaskStatus{
				// TaskUnreachable implies that the node was not reachable.
				State: execution.TaskKilled, // TODO(irvinlim): Consider adding additional state for this
				// Use Killed state since we are trying to kill it.
				Result: job.GetResultPtr(execution.JobResultKilled),
				// Explain that this task was forcefully deleted.
				Reason:  "ForceDeleted",
				Message: "Forcefully deleted the task, container may still be running",
			}

			// Set unschedulable reason if was marked to be killed by pending timeout.
			if killedFromPending := task.GetKilledFromPendingTimeoutMarker(); killedFromPending {
				newRef.DeletedStatus.Result = job.GetResultPtr(execution.JobResultPendingTimeout)
			}
		}

		newRefs = append(newRefs, *newRef)
	}
	newRj.Status.Tasks = newRefs

	// Compute new task status.
	newRj = job.UpdateJobTaskRefs(newRj, tasks)

	// Force delete the tasks.
	if err := w.deleteTasks(ctx, newRj, needDelete, true); err != nil {
		return newRj, err
	}

	return newRj, nil
}

// handleTTLAfterFinished deletes the Job if its TTLSecondsAfterFinished is exceeded.
func (w *Reconciler) handleTTLAfterFinished(
	ctx context.Context,
	rj *execution.Job,
	cfg *configv1alpha1.JobExecutionConfig,
) error {
	ttl := job.GetTTLAfterFinished(rj, cfg)

	// Skip if already being deleted.
	if isDeleted(rj) {
		return nil
	}

	// Not finished.
	if rj.Status.Condition.Finished == nil {
		return nil
	}

	// Not yet expired.
	if rj.Status.Condition.Finished.FinishedAt.Add(ttl).After(ktime.Now().Time) {
		return nil
	}

	klog.V(2).InfoS("jobcontroller: job ttl expired",
		"worker", w.Name(),
		"namespace", rj.GetNamespace(),
		"name", rj.GetName(),
		"ttl", ttl,
		"ageSinceFinish", time.Since(rj.Status.Condition.Finished.FinishedAt.Time),
	)

	// Delete this job.
	return w.client.DeleteJob(ctx, rj, metav1.DeleteOptions{})
}

// handleFinishFinalizer deletes dependent objects if it is due to be deleted.
// The purpose of this finalizer is to serialize the following:
// 1. Delete dependent tasks
// 2. Update FinishStatus if nil
// 3. Remove finalizer
// This is so that FinishStatus will always be non-nil and pods are terminated by the time they are finalized.
func (w *Reconciler) handleFinishFinalizer(
	ctx context.Context, rj *execution.Job,
) (*execution.Job, error) {
	// Not being deleted.
	if rj.DeletionTimestamp.IsZero() {
		return rj, nil
	}

	// Already finalized.
	if isFinalized(rj, executiongroup.DeleteDependentsFinalizer) {
		return rj, nil
	}

	klog.V(2).InfoS("jobcontroller: job about to be finalized",
		"worker", w.Name(),
		"finalizer", executiongroup.DeleteDependentsFinalizer,
		"namespace", rj.GetNamespace(),
		"name", rj.GetName(),
	)

	taskMgr, err := w.tasks.ForJob(rj)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get task manager")
	}

	// Get list of all created tasks from cache.
	// Use CreatedTaskRefs as they are guaranteed to contain all tasks that have been created by this Job.
	tasks := make([]jobtasks.Task, 0, len(rj.Status.Tasks))
	for _, taskRef := range rj.Status.Tasks {
		task, err := taskMgr.Lister().Get(taskRef.Name)
		if kerrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return rj, errors.Wrapf(err, "cannot get task %v", taskRef.Name)
		}

		tasks = append(tasks, task)
	}

	// There are some tasks that are still not deleted, so we need to delete them.
	if len(tasks) > 0 {
		// Update DeletedStatus for tasks.
		for _, task := range tasks {
			rj = job.UpdateTaskRefDeletedStatusIfNotSet(rj, task.GetName(), execution.TaskStatus{
				State:   execution.TaskKilled,
				Result:  job.GetResultPtr(execution.JobResultKilled),
				Reason:  "JobDeleted",
				Message: "Task was killed in response to deletion of Job",
			})
		}

		// Compute new task status.
		rj = w.updateTaskRefStatus(rj, tasks)

		// Proceed to delete tasks.
		// Task deletion should trigger a sync via informer, so don't need to enqueue a sync.
		if err := w.deleteTasks(ctx, rj, tasks, false); err != nil {
			return rj, errors.Wrapf(err, "cannot delete tasks")
		}

		return rj, nil
	}

	// Once at this part, all tasks are guaranteed to have been completely deleted.
	// Update Job's task ref status first.
	rj = w.updateTaskRefStatus(rj, tasks)

	newRj := rj.DeepCopy()

	// Remove the finalizer.
	newRj.Finalizers = k8sutils.RemoveFinalizer(newRj.Finalizers, executiongroup.DeleteDependentsFinalizer)

	klog.InfoS("jobcontroller: job finalized",
		"worker", w.Name(),
		"finalizer", executiongroup.DeleteDependentsFinalizer,
		"namespace", rj.GetNamespace(),
		"name", rj.GetName(),
	)

	return newRj, nil
}

func (w *Reconciler) setTasksKillTimestamp(
	ctx context.Context, rj *execution.Job, tasks []jobtasks.Task, killTimestamp metav1.Time,
) error {
	if err := job.ConcurrentTasks(tasks, func(task jobtasks.Task) error {
		if ktime.IsTimeSetAndEarlierThanOrEqualTo(task.GetKillTimestamp(), ktime.Now().Time) {
			return nil
		}

		if err := task.SetKillTimestamp(ctx, killTimestamp.Time); err != nil {
			return err
		}

		klog.InfoS("jobcontroller: worker set kill timestamp on task",
			"worker", w.Name(),
			"namespace", rj.GetNamespace(),
			"name", rj.GetName(),
			"task", task.GetName(),
			"deadline", killTimestamp,
		)

		w.recorder.Eventf(rj, corev1.EventTypeWarning, "Killing", "Killing task %v", task.GetName())
		return nil
	}); err != nil {
		return errors.Wrapf(err, "could not set active deadline on task")
	}

	return nil
}

// deleteTasks will concurrently delete all tasks for the given Job.
// It will also optionally update its status on its TaskRefs to avoid reconciling as TASK_LOST.
func (w *Reconciler) deleteTasks(
	ctx context.Context, rj *execution.Job, task []jobtasks.Task, force bool,
) error {
	taskMgr, err := w.tasks.ForJob(rj)
	if err != nil {
		return errors.Wrapf(err, "cannot get task manager")
	}

	eventReason := "Deleted"
	if force {
		eventReason = "ForceDeleted"
	}

	if err := job.ConcurrentTasks(task, func(task jobtasks.Task) error {
		// Don't need to delete if deletionTimestamp already set and earlier, unless we are force deleting.
		if ts := task.GetDeletionTimestamp(); !force && ktime.IsTimeSetAndEarlier(ts) {
			return nil
		}

		if err := taskMgr.Client().Delete(ctx, task.GetName(), force); kerrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			return err
		}

		klog.InfoS("jobcontroller: worker deleted task",
			"worker", w.Name(),
			"namespace", rj.GetNamespace(),
			"name", rj.GetName(),
			"task", task.GetName(),
			"force", force,
		)

		w.recorder.Eventf(rj, corev1.EventTypeWarning, eventReason, "Deleted task %v", task.GetName())
		return nil
	}); err != nil {
		return errors.Wrapf(err, "could not delete tasks")
	}

	return nil
}

// enqueueAfter will defer a sync after the specified duration, and logs the purpose of deferring
// the sync for debugging purposes.
// We enforce a lower bound of 1 second to the next sync, to slow down unwanted bursts of syncs.
func (w *Reconciler) enqueueAfter(rj *execution.Job, purpose string, duration time.Duration) {
	duration = timeutil.DurationMax(time.Second, duration)
	if key, err := cache.MetaNamespaceKeyFunc(rj); err == nil {
		w.queue.AddAfter(key, duration)
		klog.V(2).InfoS("jobcontroller: worker enqueue sync",
			"worker", w.Name(),
			"namespace", rj.GetNamespace(),
			"name", rj.GetName(),
			"purpose", purpose,
			"after", duration.String(),
		)
	}
}
