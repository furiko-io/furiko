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

package mock

import (
	"log"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
)

type Stores struct {
	controllercontext.Stores
	ctrlContext *Context
}

func NewStores(c *Context) *Stores {
	return &Stores{
		ctrlContext: c,
		Stores:      controllercontext.NewContextStores(),
	}
}

type StoreFactory interface {
	Name() string
	New(ctx controllercontext.Context) (controllermanager.Store, error)
}

func (s *Stores) RegisterFromFactoriesOrDie(factories ...StoreFactory) {
	for _, factory := range factories {
		store, err := factory.New(s.ctrlContext)
		if err != nil {
			log.Panicf("cannot create new store %v", factory.Name())
		}
		s.Register(store)
	}
}
