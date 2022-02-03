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

	"k8s.io/client-go/informers"

	furiko "github.com/furiko-io/furiko/pkg/generated/informers/externalversions"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

type Informers struct {
	kubernetes informers.SharedInformerFactory
	furiko     furiko.SharedInformerFactory
}

var _ controllercontext.Informers = &Informers{}

// NewInformers returns a new mock Informers implementations that implements
// controllercontext.Informers.
func NewInformers(clientsets controllercontext.Clientsets) *Informers {
	return &Informers{
		kubernetes: informers.NewSharedInformerFactory(clientsets.Kubernetes(), 0),
		furiko:     furiko.NewSharedInformerFactory(clientsets.Furiko(), 0),
	}
}

func (i *Informers) Kubernetes() informers.SharedInformerFactory {
	return i.kubernetes
}

func (i *Informers) Furiko() furiko.SharedInformerFactory {
	return i.furiko
}

func (i *Informers) Start(ctx context.Context) error {
	i.kubernetes.Start(ctx.Done())
	i.furiko.Start(ctx.Done())
	return nil
}
