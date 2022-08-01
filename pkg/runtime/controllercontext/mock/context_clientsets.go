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
	argo "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	argofake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	furiko "github.com/furiko-io/furiko/pkg/generated/clientset/versioned"
	furikofake "github.com/furiko-io/furiko/pkg/generated/clientset/versioned/fake"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

type Clientsets struct {
	kubernetes    *fake.Clientset
	apiExtensions *apiextensionsfake.Clientset
	furiko        *furikofake.Clientset
	argo          *argofake.Clientset
}

// NewClientsets returns a new Clientsets using fake clients.
func NewClientsets() *Clientsets {
	return &Clientsets{
		kubernetes:    fake.NewSimpleClientset(),
		apiExtensions: apiextensionsfake.NewSimpleClientset(),
		furiko:        furikofake.NewSimpleClientset(),
		argo:          argofake.NewSimpleClientset(),
	}
}

var _ controllercontext.Clientsets = (*Clientsets)(nil)

func (c *Clientsets) Kubernetes() kubernetes.Interface {
	return c.KubernetesMock()
}

// KubernetesMock returns the underlying fake clientset.
func (c *Clientsets) KubernetesMock() *fake.Clientset {
	return c.kubernetes
}

func (c *Clientsets) APIExtensions() apiextensions.Interface {
	return c.APIExtensionsMock()
}

func (c *Clientsets) APIExtensionsMock() *apiextensionsfake.Clientset {
	return c.apiExtensions
}

func (c *Clientsets) Furiko() furiko.Interface {
	return c.FurikoMock()
}

// FurikoMock returns the underlying fake clientset.
func (c *Clientsets) FurikoMock() *furikofake.Clientset {
	return c.furiko
}

func (c *Clientsets) Argo() argo.Interface {
	return c.argo
}

// ArgoMock returns the underlying fake clientset.
func (c *Clientsets) ArgoMock() *argofake.Clientset {
	return c.argo
}

// ClearActions clears all actions.
func (c *Clientsets) ClearActions() {
	c.KubernetesMock().ClearActions()
	c.APIExtensionsMock().ClearActions()
	c.FurikoMock().ClearActions()
	c.ArgoMock().ClearActions()
}
