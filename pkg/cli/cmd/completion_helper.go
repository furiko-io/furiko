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

package cmd

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// CompletionHelper provides common CompletionFunc implementations.
type CompletionHelper struct{}

// ListNamespaces returns a CompletionFunc that lists all namespaces.
func (c *CompletionHelper) ListNamespaces() CompletionFunc {
	return func(ctx context.Context, ctrlContext controllercontext.Context, cmd *cobra.Command) ([]string, error) {
		client := ctrlContext.Clientsets().Kubernetes().CoreV1()
		lst, err := client.Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "cannot list namespaces")
		}
		items := make([]string, 0, len(lst.Items))
		for _, item := range lst.Items {
			items = append(items, item.Name)
		}
		return items, nil
	}
}

// ListJobs returns a CompletionFunc that lists all jobs.
func (c *CompletionHelper) ListJobs() CompletionFunc {
	return func(ctx context.Context, ctrlContext controllercontext.Context, cmd *cobra.Command) ([]string, error) {
		namespace, err := GetNamespace(cmd)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get namespace")
		}
		lst, err := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1().Jobs(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, `cannot list jobs in namespace "%v"`, namespace)
		}
		items := make([]string, 0, len(lst.Items))
		for _, item := range lst.Items {
			items = append(items, item.Name)
		}
		return items, nil
	}
}

// ListJobConfigs returns a CompletionFunc that lists all job configs.
func (c *CompletionHelper) ListJobConfigs() CompletionFunc {
	return func(ctx context.Context, ctrlContext controllercontext.Context, cmd *cobra.Command) ([]string, error) {
		namespace, err := GetNamespace(cmd)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get namespace")
		}
		lst, err := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1().JobConfigs(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, `cannot list job configs in namespace "%v"`, namespace)
		}
		items := make([]string, 0, len(lst.Items))
		for _, item := range lst.Items {
			items = append(items, item.Name)
		}
		return items, nil
	}
}

// FromSlice returns a CompletionFunc from a slice of objects.
func (c *CompletionHelper) FromSlice(slice any) CompletionFunc {
	return func(_ context.Context, _ controllercontext.Context, _ *cobra.Command) ([]string, error) {
		return sliceToStrings(slice)
	}
}

func sliceToStrings(slice any) ([]string, error) {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input is not a slice: %v", slice)
	}
	items := make([]string, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		items = append(items, fmt.Sprintf("%v", v.Index(i)))
	}
	return items, nil
}
