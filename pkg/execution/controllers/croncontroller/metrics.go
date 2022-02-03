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

package croncontroller

import (
	"time"

	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
)

// instrumentWorkerMetrics wraps a function to instrument start and ends time for the worker.
func instrumentWorkerMetrics(name string, fn func()) func() {
	return func() {
		start := time.Now()
		reconciler.ObserveWorkerStart(name)
		fn()
		reconciler.ObserveWorkerEnd(name, time.Since(start))
	}
}
