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

package heap

import (
	"container/heap"
)

// Item is a single object in priorityQueue.
type Item struct {
	// should not be updated
	name     string
	priority int

	// will be updated whenever Push, Swap or Pop is called
	index int
}

func NewItem(name string, priority int) *Item {
	return &Item{
		name:     name,
		priority: priority,
	}
}

// Name returns the name of the item.
func (i *Item) Name() string {
	return i.name
}

// Priority returns the priority of the item.
func (i *Item) Priority() int {
	return i.priority
}

// Heap is a data structure that implements a min-heap. Not thread-safe.
//
// Although we could use workqueue.DelayingInterface for this, we don't need the
// concurrency guarantees that the workqueue package provides, and we should use
// a more straightforward implementation instead.
type Heap struct {
	pq *priorityQueue
}

// New returns a new Heap.
func New(items []*Item) *Heap {
	pq := newPriorityQueue(items)
	return &Heap{
		pq: pq,
	}
}

// Push a new Item to the heap.
func (h *Heap) Push(name string, priority int) {
	heap.Push(h.pq, NewItem(name, priority))
}

// Len returns the length of the heap.
func (h *Heap) Len() int {
	return h.pq.Len()
}

// Pop an item from the heap.
func (h *Heap) Pop() *Item {
	return heap.Pop(h.pq).(*Item)
}

// Peek an item from the heap.
func (h *Heap) Peek() (*Item, bool) {
	if h.pq.Len() <= 0 {
		return nil, false
	}
	return h.pq.queue[0], true
}

// Search returns the priority of an item in the heap.
func (h *Heap) Search(name string) (int, bool) {
	item, ok := h.pq.search(name)
	if !ok {
		return 0, false
	}
	return item.priority, true
}

// Update an item's priority in the heap.
func (h *Heap) Update(name string, newPriority int) bool {
	item, ok := h.pq.search(name)
	if !ok {
		return false
	}
	item.priority = newPriority
	heap.Fix(h.pq, item.index)
	return true
}

// Delete an item from the heap.
func (h *Heap) Delete(name string) bool {
	item, ok := h.pq.search(name)
	if !ok {
		return false
	}
	heap.Remove(h.pq, item.index)
	return true
}

// priorityQueue implements heap.Interface. Should not be utilized directly.
type priorityQueue struct {
	queue []*Item

	// mapping of names to index in the pq
	names map[string]int
}

var _ heap.Interface = (*priorityQueue)(nil)

func newPriorityQueue(items []*Item) *priorityQueue {
	queue := make([]*Item, len(items))
	names := make(map[string]int, len(items))
	for i, item := range items {
		queue[i] = &Item{
			name:     item.name,
			priority: item.priority,
			index:    i,
		}
		names[item.name] = i
	}
	pq := &priorityQueue{
		queue: queue,
		names: names,
	}
	heap.Init(pq)
	return pq
}

func (pq *priorityQueue) Len() int {
	return len(pq.queue)
}

func (pq *priorityQueue) Less(i, j int) bool {
	return pq.queue[i].priority < pq.queue[j].priority
}

func (pq *priorityQueue) Swap(i, j int) {
	pq.queue[i], pq.queue[j] = pq.queue[j], pq.queue[i]
	pq.names[pq.queue[i].name], pq.names[pq.queue[j].name] = pq.names[pq.queue[j].name], pq.names[pq.queue[i].name]
	pq.queue[i].index = i
	pq.queue[j].index = j
}

func (pq *priorityQueue) Push(x any) {
	n := len(pq.queue)
	item := x.(*Item)
	item.index = n
	pq.queue = append(pq.queue, item)
	pq.names[item.name] = n
}

func (pq *priorityQueue) Pop() any {
	old := pq.queue
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	pq.queue = old[0 : n-1]
	delete(pq.names, item.name)
	return item
}

// search looks up the name amongst items in the queue.
func (pq *priorityQueue) search(name string) (*Item, bool) {
	index, ok := pq.names[name]
	if !ok {
		return nil, false
	}
	return pq.queue[index], true
}
