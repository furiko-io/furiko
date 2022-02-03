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
	"github.com/furiko-io/furiko/pkg/utils/execution/job"
)

type jobListDiffer struct {
	items     map[string]execution.Job
	activeMap map[string]struct{}
}

func newJobListDiffer(items []execution.Job) *jobListDiffer {
	list := &jobListDiffer{
		items:     make(map[string]execution.Job),
		activeMap: make(map[string]struct{}),
	}

	for _, rj := range items {
		rj := rj
		list.items[rj.Name] = rj
		if job.IsActive(&rj) {
			list.activeMap[rj.Name] = struct{}{}
		}
	}

	return list
}

func (l *jobListDiffer) GetDiff(prevActive []execution.JobReference) (
	newJobs []execution.Job, finished []execution.Job, removed []string, isDiff bool) {
	// Keep track of unseen active jobs
	unseen := make(map[string]struct{})
	for name := range l.activeMap {
		unseen[name] = struct{}{}
	}

	// Iterate through active list to diff against.
	for _, ref := range prevActive {
		// Remove from unseen map.
		delete(unseen, ref.Name)

		// Case 1: Previously active, currently still active
		if l.isActive(ref.Name) {
			continue
		}

		// Case 2: Previously active, no longer active
		job, exists := l.getJob(ref.Name)
		if !exists {
			removed = append(removed, ref.Name)
		} else {
			finished = append(finished, job)
		}
	}

	// Add unseen jobs as new.
	for name := range unseen {
		if job, ok := l.getJob(name); ok {
			newJobs = append(newJobs, job)
		}
	}

	// Update isDiff.
	isDiff = len(removed) > 0 || len(finished) > 0 || len(newJobs) > 0

	return newJobs, finished, removed, isDiff
}

func (l *jobListDiffer) getJob(name string) (execution.Job, bool) {
	j, ok := l.items[name]
	return j, ok
}

func (l *jobListDiffer) isActive(name string) bool {
	if _, ok := l.activeMap[name]; ok {
		return true
	}
	return false
}

func (l *jobListDiffer) GetActiveRefs() []execution.JobReference {
	active := make([]execution.JobReference, 0, len(l.activeMap))
	for name := range l.activeMap {
		if job, ok := l.getJob(name); ok {
			ref := execution.JobReference{Namespace: job.Namespace, Name: job.Name}
			active = append(active, ref)
		}
	}

	// Sort refs to make return value deterministic.
	sort.Slice(active, func(i, j int) bool {
		if active[i].Namespace != active[j].Namespace {
			return active[i].Namespace < active[j].Namespace
		}
		return active[i].Name < active[j].Name
	})

	return active
}
