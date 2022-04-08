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

package jobqueuecontroller

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
)

// IndependentReconciler reconciles independent Jobs that do not have any JobConfig.
type IndependentReconciler struct {
	*Context
	concurrency *configv1alpha1.Concurrency
	client      JobControlInterface
}

func NewIndependentReconciler(ctrlContext *Context, concurrency *configv1alpha1.Concurrency) *IndependentReconciler {
	reconciler := &IndependentReconciler{
		Context:     ctrlContext,
		concurrency: concurrency,
	}
	reconciler.client = NewJobControl(
		ctrlContext.Clientsets().Furiko().ExecutionV1alpha1(),
		ctrlContext.recorder,
		reconciler.Name(),
	)
	return reconciler
}

func (r *IndependentReconciler) Name() string {
	return fmt.Sprintf("%v.Reconciler", controllerName)
}

func (r *IndependentReconciler) Concurrency() int {
	return controllerutil.GetConcurrencyOrDefaultCPUFactor(r.concurrency, 4)
}

func (r *IndependentReconciler) MaxRequeues() int {
	return -1
}

func (r *IndependentReconciler) SyncOne(ctx context.Context, namespace, name string, _ int) error {
	var err error

	trace := utiltrace.New(
		"jobqueue_independent_sync",
		utiltrace.Field{Key: "namespace", Value: namespace},
		utiltrace.Field{Key: "name", Value: name},
	)
	defer trace.LogIfLong(500 * time.Millisecond)

	klog.V(2).InfoS("jobqueuecontroller: syncing job",
		"worker", r.Name(),
		"namespace", namespace,
		"name", name,
	)

	rj, err := r.jobInformer.Lister().Jobs(namespace).Get(name)
	if kerrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "cannot get job")
	}
	trace.Step("Lookup job from cache done")

	if err := r.client.StartJob(ctx, rj); err != nil {
		return errors.Wrapf(err, "cannot start job")
	}
	trace.Step("Start job done")

	return nil
}
