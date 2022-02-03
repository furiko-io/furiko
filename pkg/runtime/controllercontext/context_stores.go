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

package controllercontext

import (
	"github.com/pkg/errors"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

var (
	ErrStoreNotRegistered = errors.New("store not registered")
)

// Stores returns shared in-memory stores, and is used for dependency injection.
func (c *Context) Stores() Stores {
	return c.storeMgr
}

type Stores interface {
	Register(store Store)
	ActiveJobStore() (ActiveJobStore, error)
}

type Store interface {
	Name() string
}

type ContextStores struct {
	stores []Store
}

func NewContextStores() *ContextStores {
	return &ContextStores{}
}

// Register a new Store.
func (c *ContextStores) Register(store Store) {
	c.stores = append(c.stores, store)
}

type ActiveJobStore interface {
	CountActiveJobsForConfig(rjc *execution.JobConfig) int64
	CheckAndAdd(rj *execution.Job, oldCount int64) (success bool)
	Delete(rj *execution.Job)
}

// ActiveJobStore returns the active job store.
func (c *ContextStores) ActiveJobStore() (ActiveJobStore, error) {
	for _, store := range c.stores {
		if s, ok := store.(ActiveJobStore); ok {
			return s, nil
		}
	}
	return nil, ErrStoreNotRegistered
}
