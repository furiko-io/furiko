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

package cmd

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

// WatchAndPrint will block on the given watch.Interface, and print all results
// using the provided handler, and returns once the context is done. If the
// received object's type does not match the generic type T, it will fall
// through to other default handlers.
func WatchAndPrint[T runtime.Object](ctx context.Context, watcher watch.Interface, filter func(obj T) bool, handler func(obj T) error) error {
	for {
		select {
		case <-ctx.Done():
			// Exit if the context is canceled.
			return nil

		case result, ok := <-watcher.ResultChan():
			// Exit if the channel is closed.
			if !ok {
				return nil
			}

			// Print the object if it matches the type T.
			if obj, ok := result.Object.(T); ok {
				if filter != nil && !filter(obj) {
					continue
				}

				if err := handler(obj); err != nil {
					return err
				}

				continue
			}

			// Fall through to default logging handlers.
			switch obj := result.Object.(type) {
			case *metav1.Status:
				// Don't print Status which may contain network error information, simply write to logs.
				klog.V(1).InfoS("received status message",
					"code", obj.Code,
					"status", obj.Status,
					"reason", obj.Reason,
					"message", obj.Message,
				)

			default:
				klog.V(1).InfoS("ignoring unhandled watch event",
					"type", fmt.Sprintf("%T", obj),
					"obj", obj,
				)
			}
		}
	}
}
