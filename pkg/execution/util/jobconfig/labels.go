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

package jobconfig

import (
	"k8s.io/apimachinery/pkg/labels"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

var (
	// AnnotationKeyScheduleTime stores the time that the Job was automatically scheduled
	// by the JobConfig's schedule spec.
	AnnotationKeyScheduleTime = executiongroup.AddGroupToLabel("schedule-time")

	// LabelKeyJobConfigUID stores the UID of the JobConfig that created the Job.
	// All Jobs are expected to have this label, which is used when using label
	// selectors.
	LabelKeyJobConfigUID = executiongroup.AddGroupToLabel("job-config-uid")

	// AnnotationKeyOptionSpecHash stores the hash of the OptionSpec at the point in
	// time when a Job's optionValues are evaluated based on the JobConfig's Option.
	AnnotationKeyOptionSpecHash = executiongroup.AddGroupToLabel("option-spec-hash")
)

// LabelJobsForJobConfig returns a labels.Set that labels all Jobs for a JobConfig.
func LabelJobsForJobConfig(rjc *execution.JobConfig) labels.Set {
	return labels.Set{
		LabelKeyJobConfigUID: string(rjc.GetUID()),
	}
}
