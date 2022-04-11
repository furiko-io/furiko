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

package variablecontext

import (
	"strconv"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
)

var (
	// ContextProvider is the global Provider that can supply context variables.
	ContextProvider Provider = &defaultProvider{}
)

// Provider supplies context variables given an object. The intention is that
// variables can only be provided in different "contexts" in a Job's lifecycle.
type Provider interface {
	// GetAllPrefixes returns a list of prefixes that the provider would inject.
	GetAllPrefixes() []string

	// MakeVariablesFromJobConfig returns context variables for the "jobconfig"
	// context, which contains information about the parent JobConfig of a Job.
	MakeVariablesFromJobConfig(rjc *execution.JobConfig) map[string]string

	// MakeVariablesFromJob returns context variables for the "job" context, which
	// contains information about the Job.
	MakeVariablesFromJob(rj *execution.Job) map[string]string

	// MakeVariablesFromTask returns context variables for the "task" context, which
	// contains information about the Task.
	MakeVariablesFromTask(rj *execution.Job, task *tasks.TaskTemplate) map[string]string
}

// defaultProvider provides the default set of context variables for a vanilla
// installation of Furiko.
type defaultProvider struct{}

var _ Provider = &defaultProvider{}

func (c *defaultProvider) GetAllPrefixes() []string {
	return []string{"jobconfig.", "job.", "task."}
}

func (c *defaultProvider) MakeVariablesFromJobConfig(rjc *execution.JobConfig) map[string]string {
	subs := map[string]string{
		"jobconfig.uid":       string(rjc.GetUID()),
		"jobconfig.name":      rjc.GetName(),
		"jobconfig.namespace": rjc.GetNamespace(),
	}

	// Add ScheduleSpec-related variables.
	if spec := rjc.Spec.Schedule; spec != nil && spec.Cron != nil {
		subs["jobconfig.cron_schedule"] = spec.Cron.Expression
		subs["jobconfig.cron_timezone"] = spec.Cron.Timezone
	}

	return subs
}

func (c *defaultProvider) MakeVariablesFromJob(rj *execution.Job) map[string]string {
	subs := map[string]string{
		"job.uid":       string(rj.GetUID()),
		"job.name":      rj.GetName(),
		"job.namespace": rj.GetNamespace(),
		"job.type":      string(rj.Spec.Type),
	}

	if maxRetries := rj.Spec.Template.MaxRetryAttempts; maxRetries != nil {
		subs["job.max_retries"] = strconv.Itoa(int(*maxRetries))
	}

	return subs
}

func (c *defaultProvider) MakeVariablesFromTask(rj *execution.Job, task *tasks.TaskTemplate) map[string]string {
	return map[string]string{
		"task.name":        task.Name,
		"task.namespace":   rj.GetNamespace(), // assumes Task shares namespace with Job
		"task.retry_index": strconv.Itoa(int(task.RetryIndex)),
	}
}
