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
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// ListJobConfigsCompleter is a Completer that lists job configs.
type ListJobConfigsCompleter struct {
	// If specified, overrides the sort function.
	Sort func(a, b *execution.JobConfig) bool

	// If specified, filters out job configs that should not appear in completions.
	Filter func(job *execution.JobConfig) bool

	// If specified, overrides the text formatted in the completion.
	// Use \t to add labels for the completion values.
	Format func(job *execution.JobConfig) string
}

var _ Completer = (*ListJobConfigsCompleter)(nil)

func (c *ListJobConfigsCompleter) Complete(ctx context.Context, ctrlContext controllercontext.Context, cmd *cobra.Command) ([]string, error) {
	namespace, err := common.GetNamespace(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get namespace")
	}
	lst, err := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1().JobConfigs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, `cannot list job configs in namespace "%v"`, namespace)
	}

	items := make([]string, 0, len(lst.Items))

	sortFunc := c.defaultSort
	if c.Sort != nil {
		sortFunc = c.Sort
	}
	sort.Slice(lst.Items, func(i, j int) bool {
		return sortFunc(&lst.Items[i], &lst.Items[j])
	})

	for _, item := range lst.Items {
		item := item

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

// Sort by name in ascending order.
func (c *ListJobConfigsCompleter) defaultSort(a, b *execution.JobConfig) bool {
	return a.Name < b.Name
}

// Format to print phase and last executed time.
func (c *ListJobConfigsCompleter) defaultFormat(jobConfig *execution.JobConfig) string {
	var lastExecuted string
	if !jobConfig.Status.LastExecuted.IsZero() {
		lastExecuted = ", last executed " + formatter.FormatTimeWithTimeAgo(jobConfig.Status.LastExecuted)
	}
	return fmt.Sprintf("%v\t%v%v", jobConfig.Name, jobConfig.Status.State, lastExecuted)
}
