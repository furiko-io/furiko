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

// nolint:dupl
package completion

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/format"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// ListJobsCompleter is a Completer that lists jobs.
type ListJobsCompleter struct {
	// If specified, overrides the sort function.
	Sort func(a, b *execution.Job) bool

	// If specified, filters out jobs that should not appear in completions.
	Filter func(job *execution.Job) bool

	// If specified, overrides the text formatted in the completion.
	// Use \t to add labels for the completion values.
	Format func(job *execution.Job) string
}

var _ Completer = (*ListJobsCompleter)(nil)

func (c *ListJobsCompleter) Complete(ctx context.Context, ctrlContext controllercontext.Context, namespace string) ([]string, error) {
	lst, err := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, `cannot list jobs in namespace "%v"`, namespace)
	}

	items := make([]string, 0, len(lst.Items))

	// By default, sort jobs in descending order of creation timestamp.
	// If we sort by startTime, there's no reasonable sort order that makes sense for Queued jobs.
	sortFunc := c.defaultSort
	if c.Sort != nil {
		sortFunc = c.Sort
	}
	sort.Slice(lst.Items, func(i, j int) bool {
		return sortFunc(&lst.Items[i], &lst.Items[j])
	})

	for _, item := range lst.Items {
		// Filter item.
		if c.Filter != nil && !c.Filter(&item) {
			continue
		}

		// Format item.
		formatFunc := c.defaultFormat
		if c.Format != nil {
			formatFunc = c.Format
		}

		items = append(items, formatFunc(&item))
	}

	return items, nil
}

// Sort by creation timestamp in descending order.
func (c *ListJobsCompleter) defaultSort(a, b *execution.Job) bool {
	return a.CreationTimestamp.After(b.CreationTimestamp.Time)
}

// Format to print phase and creationTimestamp as part of the completion label.
func (c *ListJobsCompleter) defaultFormat(job *execution.Job) string {
	var timestamp string

	if finishCondition := job.Status.Condition.Finished; finishCondition != nil && !finishCondition.FinishTimestamp.IsZero() {
		timestamp = fmt.Sprintf("finished %v", format.TimeWithTimeAgo(&finishCondition.FinishTimestamp))
	} else if !job.Status.StartTime.IsZero() {
		timestamp = fmt.Sprintf("started %v", format.TimeWithTimeAgo(job.Status.StartTime))
	} else {
		timestamp = fmt.Sprintf("created %v", format.TimeWithTimeAgo(&job.CreationTimestamp))
	}

	return fmt.Sprintf("%v\t%v, %v", job.Name, job.Status.Phase, timestamp)
}
