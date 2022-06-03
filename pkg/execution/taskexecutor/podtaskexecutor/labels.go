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
	"k8s.io/apimachinery/pkg/labels"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

var (
	// LabelKeyJobUID label is added on Pods to identify the parent Job by UID.
	LabelKeyJobUID = executiongroup.AddGroupToLabel("job-uid")

	// LabelKeyTaskRetryIndex label is added on Pods to indicate the retry index of
	// the task.
	LabelKeyTaskRetryIndex = executiongroup.AddGroupToLabel("task-retry-index")

	// LabelKeyTaskParallelIndexHash label is added on Pods to indicate the parallel
	// index hash of the task.
	LabelKeyTaskParallelIndexHash = executiongroup.AddGroupToLabel("task-parallel-index-hash")

	// AnnotationKeyTaskParallelIndex annotations is added on Pods to indicate the parallel
	// index of the task in JSON format.
	AnnotationKeyTaskParallelIndex = executiongroup.AddGroupToLabel("task-parallel-index")
)

// LabelPodsForJob returns a labels.Set that labels all Pods for a Job.
func LabelPodsForJob(rj *execution.Job) labels.Set {
	return labels.Set{
		LabelKeyJobUID: string(rj.GetUID()),
	}
}
