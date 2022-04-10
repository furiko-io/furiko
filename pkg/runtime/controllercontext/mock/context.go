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
	"context"

	"github.com/pkg/errors"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

// Context is a mock context that implements controllercontext.Context and is
// made up of mock components that can be used for unit tests.
type Context struct {
	clientsets *Clientsets
	configs    *Configs
	informers  controllercontext.Informers
	stores     controllercontext.Stores
}

var _ controllercontext.Context = &Context{}

func NewContext() *Context {
	clientsets := NewClientsets()
	return &Context{
		clientsets: clientsets,
		configs:    NewConfigs(),
		informers:  NewInformers(clientsets),
		stores:     controllercontext.NewContextStores(), // not mocked
	}
}

func (c *Context) Clientsets() controllercontext.Clientsets {
	return c.MockClientsets()
}

// MockClientsets returns the underlying mock clientsets.
func (c *Context) MockClientsets() *Clientsets {
	return c.clientsets
}

func (c *Context) Configs() controllercontext.Configs {
	return c.MockConfigs()
}

func (c *Context) MockConfigs() *Configs {
	return c.configs
}

func (c *Context) Informers() controllercontext.Informers {
	return c.informers
}

func (c *Context) Stores() controllercontext.Stores {
	return c.stores
}

func (c *Context) Start(ctx context.Context) error {
	// Start config manager.
	if err := c.configs.Start(ctx); err != nil {
		return errors.Wrapf(err, "cannot start dynamic config manager")
	}

	// Start informers.
	if err := c.informers.Start(ctx); err != nil {
		return errors.Wrapf(err, "cannot start informers")
	}

	return nil
}
