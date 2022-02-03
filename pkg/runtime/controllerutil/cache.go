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

package controllerutil

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/tools/cache"
)

var (
	ErrWaitForCacheSyncTimeout = errors.New("timed out while waiting for cache to sync")
)

// WaitForNamedCacheSyncWithTimeout is a wrapper around WaitForNamedCacheSync
// subject to a maximum timeout. Returns whether there was a successful or
// failed sync.
func WaitForNamedCacheSyncWithTimeout(
	ctx context.Context, controllerName string, timeout time.Duration, cacheSyncs ...cache.InformerSynced,
) error {
	timer := time.NewTimer(timeout)
	timeoutCh := make(chan struct{})

	go func() {
		select {
		case <-timer.C: // If deadline was exceeded
		case <-ctx.Done(): // If controller is asked to stop
		}

		timeoutCh <- struct{}{}
	}()

	if !cache.WaitForNamedCacheSync(controllerName, timeoutCh, cacheSyncs...) {
		return ErrWaitForCacheSyncTimeout
	}

	return nil
}
