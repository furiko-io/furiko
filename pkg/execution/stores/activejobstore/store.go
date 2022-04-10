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

package activejobstore

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	utilatomic "github.com/furiko-io/furiko/pkg/utils/atomic"
	"github.com/furiko-io/furiko/pkg/utils/execution/job"
	"github.com/furiko-io/furiko/pkg/utils/execution/jobconfig"
)

var (
	PanicOnInvariantViolation bool
)

// Store is an in-memory store of active Job counts per JobConfig.
//
// This store is expected to be recovered prior to starting the controller, and
// helps to guard against multiple concurrent jobs for the same JobConfig from
// being started in an atomic fashion.
type Store struct {
	counter   *utilatomic.Counter
	recovered bool
	mu        sync.Mutex

	informer *InformerWorker
}

func NewStore(ctrlContext controllercontext.Context) (*Store, error) {
	store := &Store{
		counter: utilatomic.NewCounter(),
	}
	store.informer = NewInformerWorker(ctrlContext, store)
	return store, nil
}

func (s *Store) Name() string {
	return storeName
}

// CountActiveJobsForConfig returns the count of active Jobs for a JobConfig.
func (s *Store) CountActiveJobsForConfig(rjc *execution.JobConfig) int64 {
	return s.counter.Get(string(rjc.UID))
}

// CheckAndAdd atomically increments the active count for the given JobConfig.
func (s *Store) CheckAndAdd(rjc *execution.JobConfig, oldCount int64) bool {
	key := string(rjc.UID)
	res := s.counter.CheckAndAdd(key, oldCount)
	if res {
		klog.V(5).InfoS("activejobstore: incremented job counter", "key", key, "count", oldCount+1)
	}
	return res
}

// Delete the Job from the store.
func (s *Store) Delete(rjc *execution.JobConfig) {
	key := string(rjc.UID)
	count := s.decrement(key)
	klog.V(5).InfoS("activejobstore: decremented job counter", "key", key, "count", count)
}

// Recover should be called after the context is started and should be called only once.
func (s *Store) Recover(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.recovered {
		return errors.New("already recovered previously")
	}

	// Start informer.
	s.informer.Start(ctx.Done())

	// Wait for cache to sync.
	if err := controllerutil.WaitForNamedCacheSyncWithTimeout(ctx, s.Name(), time.Minute*3,
		s.informer.HasSynced); err != nil {
		klog.ErrorS(err, "activejobstore: cache sync timeout")
		return err
	}

	// List all jobs.
	jobs, err := s.informer.jobInformer.Lister().List(labels.Everything())
	if err != nil {
		return errors.Wrapf(err, "cannot list jobs")
	}

	// Add all active jobs to the store.
	for _, rj := range jobs {
		key, ok := s.getKey(rj)
		if ok && job.IsActive(rj) {
			s.increment(key)
		}
	}

	s.recovered = true
	return nil
}

func (s *Store) OnUpdate(oldRj, newRj *execution.Job) {
	// Assume that label is not removed.
	key, ok := s.getKey(oldRj)
	if !ok {
		return
	}

	// A Job transitioned from active to inactive.
	if job.IsActive(oldRj) && !job.IsActive(newRj) {
		count := s.decrement(key)
		klog.V(5).InfoS("activejobstore: job became inactive", "key", key, "count", count)
		return
	}

	// A Job transitioned from inactive to active. Note that we have to avoid
	// double-incrementing if a job was previously unstarted and was now started,
	// since we expect to use CheckAndAdd to do so.
	if !job.IsActive(oldRj) && job.IsActive(newRj) &&
		!(!job.IsStarted(oldRj) && job.IsStarted(newRj)) {
		count := s.increment(key)
		klog.V(5).InfoS("activejobstore: job became active", "key", key, "count", count)
		return
	}
}

func (s *Store) OnDelete(rj *execution.Job) {
	key, ok := s.getKey(rj)
	if !ok {
		return
	}

	// Only need to decrement counter if it was active.
	if job.IsActive(rj) {
		count := s.decrement(key)
		klog.V(5).InfoS("activejobstore: active job was deleted", "key", key, "count", count)
	}
}

func (s *Store) increment(key string) int64 {
	count := s.counter.Add(key)
	if count > 1 {
		err := errors.New("job counter more than 1")
		klog.ErrorS(err, "activejobstore: invariant violation", "key", key, "count", count)
		if PanicOnInvariantViolation {
			log.Panicf("activejobstore: invariant violation: %v key=%v count=%v", err, key, count)
		}
	}
	return count
}

func (s *Store) decrement(key string) int64 {
	count := s.counter.Remove(key)
	if count < 0 {
		err := errors.New("job counter less than 0")
		klog.ErrorS(err, "activejobstore: invariant violation", "key", key, "count", count)
		if PanicOnInvariantViolation {
			log.Panicf("activejobstore: invariant violation: %v key=%v count=%v", err, key, count)
		}
	}
	return count
}

// getKey returns the topology key for counting Jobs. In our case, it is the UID
// of the parent JobConfig. In case the label does not exist, callers should
// ignore the store if false is returned.
func (s *Store) getKey(rj *execution.Job) (string, bool) {
	uid, ok := rj.Labels[jobconfig.LabelKeyJobConfigUID]
	return uid, ok
}
