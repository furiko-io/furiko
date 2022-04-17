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

package jobconfigcontroller

import (
	"sort"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/cmp"
)

type FilterFunc func(*execution.Job) bool

// IsJobConfigStatusEqual returns true if the JobConfigStatus is not equal for a JobConfig.
func IsJobConfigStatusEqual(orig, updated *execution.JobConfig) (bool, error) {
	newUpdated := orig.DeepCopy()
	newUpdated.Status = updated.Status
	return cmp.IsJSONEqual(orig, newUpdated)
}

// FilterJobs filters a list of Job.
func FilterJobs(items []*execution.Job, filterFunc FilterFunc) []*execution.Job {
	filtered := make([]*execution.Job, 0, len(items))
	for _, item := range items {
		if filterFunc(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// ToJobReferences converts a list of Jobs to a list of JobReferences.
func ToJobReferences(items []*execution.Job) []execution.JobReference {
	refs := make([]execution.JobReference, 0, len(items))
	for _, item := range items {
		ref := execution.JobReference{
			UID:               item.GetUID(),
			Name:              item.Name,
			CreationTimestamp: item.CreationTimestamp,
			Phase:             item.Status.Phase,
		}
		if !item.Status.StartTime.IsZero() {
			ref.StartTime = item.Status.StartTime.DeepCopy()
		}
		refs = append(refs, ref)
	}

	// Sort refs to make return value deterministic.
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].CreationTimestamp.Before(&refs[j].CreationTimestamp)
	})

	return refs
}
