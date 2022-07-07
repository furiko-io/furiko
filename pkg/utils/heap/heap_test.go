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

package heap_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/furiko-io/furiko/pkg/utils/heap"
)

// ExampleHeap implements the test from container/heap.
func ExampleHeap() {
	hp := heap.New([]*heap.Item{
		heap.NewItem("banana", 3),
		heap.NewItem("apple", 2),
		heap.NewItem("pear", 4),
	})

	// Insert a new item and then modify its priority.
	hp.Push("orange", 1)
	hp.Update("orange", 5)

	// Take the items out; they arrive in increasing priority order.
	for hp.Len() > 0 {
		item := hp.Pop()
		fmt.Printf("%.2d:%s ", item.Priority(), item.Name())
	}
	// Output:
	// 02:apple 03:banana 04:pear 05:orange
}

func TestHeap(t *testing.T) {
	hp := heap.New([]*heap.Item{
		heap.NewItem("banana", 3),
		heap.NewItem("apple", 2),
		heap.NewItem("pear", 4),
	})

	assert.Equal(t, 3, hp.Len())

	// Peek the first item
	item, ok := hp.Peek()
	assert.True(t, ok)
	assert.Equal(t, "apple", item.Name())
	assert.Equal(t, 2, item.Priority())

	// Delete an item
	assert.True(t, hp.Delete("apple"))
	assert.Equal(t, 2, hp.Len())
	item, ok = hp.Peek()
	assert.True(t, ok)
	assert.Equal(t, "banana", item.Name())
	assert.Equal(t, 3, item.Priority())

	// Cannot delete more than once
	assert.False(t, hp.Delete("apple"))
	assert.Equal(t, 2, hp.Len())

	// Cannot update the item after deleted
	assert.False(t, hp.Update("apple", 404))
	assert.Equal(t, 2, hp.Len())

	// Add a new item
	hp.Push("mango", 42)
	assert.Equal(t, 3, hp.Len())
	item, ok = hp.Peek()
	assert.True(t, ok)
	assert.Equal(t, "banana", item.Name())
	assert.Equal(t, 3, item.Priority())

	// Update the item
	assert.True(t, hp.Update("mango", 0))
	assert.Equal(t, 3, hp.Len())
	item, ok = hp.Peek()
	assert.True(t, ok)
	assert.Equal(t, "mango", item.Name())
	assert.Equal(t, 0, item.Priority())

	// Pop the added item
	item = hp.Pop()
	assert.Equal(t, 2, hp.Len())
	assert.Equal(t, "mango", item.Name())
	assert.Equal(t, 0, item.Priority())

	// Peek the next item
	item, ok = hp.Peek()
	assert.True(t, ok)
	assert.Equal(t, "banana", item.Name())
	assert.Equal(t, 3, item.Priority())

	// Cannot remove the item after popped
	assert.False(t, hp.Delete("mango"))
	assert.Equal(t, 2, hp.Len())

	// Cannot update the item after popped
	assert.False(t, hp.Update("mango", 404))
	assert.Equal(t, 2, hp.Len())

	// Pop everything
	for hp.Len() > 0 {
		hp.Pop()
	}

	// Nothing left to peek
	_, ok = hp.Peek()
	assert.False(t, ok)
}
