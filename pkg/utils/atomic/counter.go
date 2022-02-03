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

package atomic

import (
	"sync/atomic"
)

type counterNode struct {
	count int64
}

// Counter keeps track of counts by a given key.
type Counter struct {
	countMap Map
}

func NewCounter() *Counter {
	return &Counter{}
}

func (a *Counter) getNode(key string, autoCreate bool) *counterNode {
	raw, exists := a.countMap.Load(key)
	if exists {
		return raw.(*counterNode)
	}
	var candidate *counterNode
	if autoCreate {
		candidate = &counterNode{}
		raw, exists = a.countMap.LoadOrStore(key, candidate)
		if exists {
			return raw.(*counterNode)
		}
	}
	return candidate
}

func (a *Counter) Add(key string) int64 {
	node := a.getNode(key, true)
	return atomic.AddInt64(&node.count, 1)
}

func (a *Counter) Remove(key string) int64 {
	node := a.getNode(key, true)
	return atomic.AddInt64(&node.count, -1)
}

func (a *Counter) Get(key string) int64 {
	node := a.getNode(key, false)
	if node == nil {
		return 0
	}
	return atomic.LoadInt64(&node.count)
}

// CheckAndAdd uses CompareAndSwap to increment the count, only if the old count matches.
// Returns false if the CompareAndSwap failed.
func (a *Counter) CheckAndAdd(key string, old int64) bool {
	node := a.getNode(key, true)
	return atomic.CompareAndSwapInt64(&node.count, old, old+1)
}

func (a *Counter) Clear() {
	a.countMap.Clear()
}
