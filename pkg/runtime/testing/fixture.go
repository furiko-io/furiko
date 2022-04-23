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

package testing

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// InitFixtures initializes a set of fixtures against a clientset.
func InitFixtures(ctx context.Context, client controllercontext.Clientsets, fixtures []runtime.Object) error {
	for _, fixture := range fixtures {
		if err := InitFixture(ctx, client, fixture); err != nil {
			return errors.Wrapf(err, "cannot initialize fixture %v", fixture)
		}
	}

	return nil
}

// InitFixture initializes a single fixture against a clientset.
// TODO(irvinlim): Currently we just hardcode a list of types to be initialized,
//  we could probably use reflection instead.
func InitFixture(ctx context.Context, client controllercontext.Clientsets, fixture runtime.Object) error {
	var err error
	switch f := fixture.(type) {
	case *corev1.Pod:
		_, err = client.Kubernetes().CoreV1().Pods(f.Namespace).Create(ctx, f, metav1.CreateOptions{})
	case *execution.Job:
		_, err = client.Furiko().ExecutionV1alpha1().Jobs(f.Namespace).Create(ctx, f, metav1.CreateOptions{})
	case *execution.JobConfig:
		_, err = client.Furiko().ExecutionV1alpha1().JobConfigs(f.Namespace).Create(ctx, f, metav1.CreateOptions{})
	}
	return err
}
