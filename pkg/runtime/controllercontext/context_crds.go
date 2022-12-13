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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/client/listers/apiextensions/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
)

func (c *ctrlContext) CustomResourceDefinitions() CRDs {
	return c.crdMgr
}

type CRDs interface {
	Start(ctx context.Context) error
	ArgoWorkflows() (*apiextensionsv1.CustomResourceDefinition, error)
}

type contextCRDs struct {
	informers Informers
	hasSynced cache.InformerSynced
	crdLister v1.CustomResourceDefinitionLister
}

var _ CRDs = (*contextCRDs)(nil)

func SetUpCRDs(informers Informers) CRDs {
	crdClient := informers.APIExtensions().Apiextensions().V1().CustomResourceDefinitions()
	return &contextCRDs{
		informers: informers,
		hasSynced: crdClient.Informer().HasSynced,
		crdLister: crdClient.Lister(),
	}
}

func (c *contextCRDs) Start(ctx context.Context) error {
	// Start APIExtensions informer and wait for sync. This is required to assess
	// the availability of CustomResourceDefinitions that are installed in the
	// cluster.
	c.informers.APIExtensions().Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), c.hasSynced) {
		return errors.Wrapf(controllerutil.ErrWaitForCacheSyncTimeout, "cannot sync customresourcedefinitions")
	}

	return nil
}

func (c *contextCRDs) ArgoWorkflows() (*apiextensionsv1.CustomResourceDefinition, error) {
	return c.crdLister.Get("workflows.argoproj.io")
}
