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

package completion_test

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/furiko-io/furiko/pkg/cli/completion"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
)

var (
	defaultNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultNamespace,
			Labels: map[string]string{
				"description": "The default namespace",
			},
		},
	}

	prodNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ProdNamespace,
			Labels: map[string]string{
				"description": "For prod jobs",
			},
		},
	}
)

func TestListNamespacesCompleter(t *testing.T) {
	runtimetesting.RunCompleterTests(t, []runtimetesting.CompleterTest{
		{
			Name:      "no namespaces",
			Completer: &completion.ListNamespacesCompleter{},
			Want:      []string{},
		},
		{
			Name: "show namespaces",
			Fixtures: []runtime.Object{
				defaultNamespace,
				prodNamespace,
			},
			Completer: &completion.ListNamespacesCompleter{},
			Want: []string{
				DefaultNamespace,
				ProdNamespace,
			},
		},
		{
			Name: "with sort",
			Fixtures: []runtime.Object{
				defaultNamespace,
				prodNamespace,
			},
			Completer: &completion.ListNamespacesCompleter{
				Sort: func(a, b *corev1.Namespace) bool {
					return a.Name > b.Name
				},
			},
			Want: []string{
				ProdNamespace,
				DefaultNamespace,
			},
		},
		{
			Name: "with filter",
			Fixtures: []runtime.Object{
				defaultNamespace,
				prodNamespace,
			},
			Completer: &completion.ListNamespacesCompleter{
				Filter: func(ns *corev1.Namespace) bool {
					return ns.Name == DefaultNamespace
				},
			},
			Want: []string{
				DefaultNamespace,
			},
		},
		{
			Name: "with format",
			Fixtures: []runtime.Object{
				defaultNamespace,
				prodNamespace,
			},
			Completer: &completion.ListNamespacesCompleter{
				Format: func(ns *corev1.Namespace) string {
					return fmt.Sprintf("%v\t%v", ns.Name, ns.Labels["description"])
				},
			},
			Want: []string{
				`default	The default namespace`,
				`prod	For prod jobs`,
			},
		},
	})
}
