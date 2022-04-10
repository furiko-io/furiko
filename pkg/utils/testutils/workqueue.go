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

package testutils

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/workqueue"
)

// WorkqueueGetWithTimeout gets from a workqueue with a timeout.
func WorkqueueGetWithTimeout(queue workqueue.Interface, timeout time.Duration) (interface{}, bool, error) {
	type result struct {
		item     interface{}
		shutdown bool
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultChan := make(chan result)
	go func() {
		item, shutdown := queue.Get()
		resultChan <- result{
			item:     item,
			shutdown: shutdown,
		}
	}()

	select {
	case res := <-resultChan:
		return res.item, res.shutdown, nil
	case <-ctx.Done():
		return nil, false, errors.New("workqueue get timeout")
	}
}
