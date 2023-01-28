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

package completion

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// ListNamespacesCompleter is a Completer that lists job configs.
type ListNamespacesCompleter struct {
	// If specified, overrides the sort function.
	Sort func(a, b *corev1.Namespace) bool

	// If specified, filters out job configs that should not appear in completions.
	Filter func(job *corev1.Namespace) bool

	// If specified, overrides the text formatted in the completion.
	// Use \t to add labels for the completion values.
	Format func(job *corev1.Namespace) string
}

var _ Completer = (*ListNamespacesCompleter)(nil)

func (c *ListNamespacesCompleter) Complete(ctx context.Context, ctrlContext controllercontext.Context, cmd *cobra.Command) ([]string, error) {
	client := ctrlContext.Clientsets().Kubernetes().CoreV1()
	lst, err := client.Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot list namespaces")
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
func (c *ListNamespacesCompleter) defaultSort(a, b *corev1.Namespace) bool {
	return a.Name < b.Name
}

// Format to print phase and last executed time.
func (c *ListNamespacesCompleter) defaultFormat(namespace *corev1.Namespace) string {
	return namespace.Name
}
