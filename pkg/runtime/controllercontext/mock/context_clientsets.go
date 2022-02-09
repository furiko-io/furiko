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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	furiko "github.com/furiko-io/furiko/pkg/generated/clientset/versioned"
	furikofake "github.com/furiko-io/furiko/pkg/generated/clientset/versioned/fake"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

type Clientsets struct {
	kubernetes *fake.Clientset
	furiko     *furikofake.Clientset
}

// NewClientsets returns a new Clientsets using fake clients.
func NewClientsets() *Clientsets {
	return &Clientsets{
		kubernetes: fake.NewSimpleClientset(),
		furiko:     furikofake.NewSimpleClientset(),
	}
}

var _ controllercontext.Clientsets = &Clientsets{}

func (c *Clientsets) Kubernetes() kubernetes.Interface {
	return c.KubernetesMock()
}

// KubernetesMock returns the underlying fake clientset.
func (c *Clientsets) KubernetesMock() *fake.Clientset {
	return c.kubernetes
}

func (c *Clientsets) Furiko() furiko.Interface {
	return c.FurikoMock()
}

// FurikoMock returns the underlying fake clientset.
func (c *Clientsets) FurikoMock() *furikofake.Clientset {
	return c.furiko
}

// ClearActions clears all actions.
func (c *Clientsets) ClearActions() {
	c.KubernetesMock().ClearActions()
	c.FurikoMock().ClearActions()
}
