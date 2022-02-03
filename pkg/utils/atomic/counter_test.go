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

package atomic_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/furiko-io/furiko/pkg/utils/atomic"
)

func TestCheckAndAdd(t *testing.T) {
	assert := assert.New(t)
	t.Run("detect TOCTTOU among multiple goroutines", func(t *testing.T) {
		c := atomic.NewCounter()

		read1 := make(chan struct{})
		var read2 sync.WaitGroup
		var wg sync.WaitGroup
		key := "1"
		numConcurrentWriters := 100

		read2.Add(numConcurrentWriters - 1)
		wg.Add(numConcurrentWriters - 1)
		go func() {
			old := c.Get(key)
			assert.Equal(int64(0), old, "first read should be 0")
			read2.Wait()
			assert.True(c.CheckAndAdd(key, old), "first update must succeed")
			close(read1)
		}()

		for i := 0; i < numConcurrentWriters-1; i++ {
			go func() {
				old := c.Get(key)
				assert.Equal(int64(0), old, "first read should be 0")
				read2.Done()
				<-read1
				assert.False(c.CheckAndAdd(key, old), "second update must fail")
				wg.Done()
			}()
		}
		wg.Wait()
	})
}
