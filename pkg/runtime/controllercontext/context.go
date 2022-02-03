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

package controllercontext

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
)

// Context is a shared controller context that can be safely shared between controllers.
type Context struct {
	restConfig *rest.Config
	configMgr  Configs
	storeMgr   Stores
	clientsets Clientsets
	informers  Informers
}

var _ ContextInterface = &Context{}

// NewForConfig prepares a new Context from a kubeconfig and controller manager config spec.
func NewForConfig(cfg *rest.Config, ctrlConfig *configv1.BootstrapConfigSpec) (*Context, error) {
	ctrlContext := &Context{
		restConfig: cfg,
	}

	// Set up clientsets.
	clientsets, err := SetUpClientsets(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot set up clientsets")
	}
	ctrlContext.clientsets = clientsets

	// Set up shared informer factories.
	ctrlContext.informers = SetUpInformers(ctrlContext.clientsets, ctrlConfig)

	// Set up config manager.
	ctrlContext.configMgr = SetUpConfigManager(ctrlConfig, ctrlContext.Clientsets().Kubernetes())

	// Set up stores.
	ctrlContext.storeMgr = NewContextStores()

	return ctrlContext, nil
}

func (c *Context) Start(ctx context.Context) error {
	// Start config manager.
	if err := c.configMgr.Start(ctx); err != nil {
		return errors.Wrapf(err, "cannot start dynamic config manager")
	}

	// Start informers.
	if err := c.informers.Start(ctx); err != nil {
		return errors.Wrapf(err, "cannot start informers")
	}

	return nil
}
