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

package reconciler

import (
	"context"
	"fmt"
	"sync"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Reconciler is implemented by reconciliation-based controllers.
type Reconciler interface {
	Name() string
	Concurrency() int
	MaxRequeues() int
	SyncOne(ctx context.Context, namespace, name string, numRequeues int) error
}

// Controller is a reconciler controller template that handles concurrency,
// workqueue and retries with a Reconciler handler.
type Controller struct {
	handler Reconciler
	queue   workqueue.RateLimitingInterface
	wg      sync.WaitGroup

	// SplitMetaNamespaceKey is the function used to split a key into namespace and name.
	// Defaults to cache.SplitMetaNamespaceKey.
	SplitMetaNamespaceKey func(string) (string, string, error)
}

// NewController creates a Controller that uses the given Reconciler to handle
// syncs from a rate-limited workqueue.
func NewController(handler Reconciler, queue workqueue.RateLimitingInterface) *Controller {
	return &Controller{
		handler: handler,
		queue:   queue,

		SplitMetaNamespaceKey: cache.SplitMetaNamespaceKey,
	}
}

func (w *Controller) Start(ctx context.Context) {
	concurrency := w.handler.Concurrency()
	w.wg.Add(concurrency)

	// Start workers in the background.
	for i := 0; i < concurrency; i++ {
		go func() {
			defer w.wg.Done()
			wait.UntilWithContext(ctx, w.worker, time.Second)
		}()
	}

	ObserveWorkersTotal(w.handler.Name(), w.handler.Concurrency())
}

// Wait until all reconciler workers have exited. They should exit once they are
// done with their work, and a Get from the workqueue is telling them to shut
// down.
func (w *Controller) Wait() {
	w.wg.Wait()
}

func (w *Controller) worker(ctx context.Context) {
	// Perform work until told to quit.
	for w.work(ctx) {
	}
}

func (w *Controller) work(ctx context.Context) bool {
	// Fetch item from queue. This blocks when there is nothing in the queue.
	item, quit := w.queue.Get()
	if quit {
		return false
	}

	// Call Done so that processing can take place for the key again after return.
	defer w.queue.Done(item)

	// Process a single item from the workqueue.
	err := w.syncItem(ctx, item)

	// Handle error for logs and metrics.
	if err != nil {
		w.handleError(err, item)
		return true
	}

	// Reset requeues if successful.
	w.queue.Forget(item)
	return true
}

// syncItem processes a single item from the workqueue.
func (w *Controller) syncItem(ctx context.Context, item interface{}) error {
	// Type assert key.
	key, ok := item.(string)
	if !ok {
		return fmt.Errorf("unexpected type, expected string got type %T", item)
	}

	// Split namespace/name
	namespace, name, err := w.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	// Process job.
	err = instrumentMetrics(w.handler.Name(), func() error {
		return w.handler.SyncOne(ctx, namespace, name, w.queue.NumRequeues(key))
	})

	// Ignore errors when it is created successfully.
	if err == nil {
		return nil
	}

	// Otherwise, requeue item to create later.
	if w.handler.MaxRequeues() <= 0 || w.queue.NumRequeues(key) < w.handler.MaxRequeues() {
		w.queue.AddRateLimited(key)
	}

	// Maximum retries exceeded, will not automatically retry.
	// Any retries after this has to come from informers.

	// Log exactly at the point when we first exceeded max requeues.
	if w.queue.NumRequeues(key) == w.handler.MaxRequeues() {
		ObserveRetriesExceeded(w.handler.Name())
		klog.Error("reconciler: sync retries exceeded",
			"worker", w.handler.Name(), "namespace", namespace, "name", name)
	}

	return err
}

// handleError handles graceful logging of errors and differentiated observation
// of metrics. We choose to print errors differently depending on the error
// cause, since some of them are retryable and hence ignorable, do not print to
// logs.
func (w *Controller) handleError(err error, item interface{}) {
	switch {
	case kerrors.IsConflict(err):
		// Drop to verbose error.
		klog.V(3).ErrorS(err, "reconciler: sync item conflict",
			"worker", w.handler.Name(), "item", item)
		ObserveControllerSyncError(w.handler.Name(), "Conflict")

	default:
		// Show default error log message.
		klog.ErrorS(err, "reconciler: sync item error",
			"worker", w.handler.Name(), "item", item)
		ObserveControllerSyncError(w.handler.Name(), "")
	}
}
