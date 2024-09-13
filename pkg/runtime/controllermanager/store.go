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

package controllermanager

import (
	"context"

	"golang.org/x/sync/errgroup"
)

type Store interface {
	Name() string
	Recover(ctx context.Context) error
}

// RecoverStores recovers all stores in the background and blocks until all of them have finished recovering.
func RecoverStores(ctx context.Context, stores []Store) error {
	// Don't use derived context, as it will be canceled once Wait returns.
	group, _ := errgroup.WithContext(ctx)

	for _, store := range stores {
		group.Go(func() error {
			return store.Recover(ctx)
		})
	}

	return group.Wait()
}
