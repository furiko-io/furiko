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
	argo "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	furiko "github.com/furiko-io/furiko/pkg/generated/clientset/versioned"
)

func (c *ctrlContext) Clientsets() Clientsets {
	return c.clientsets
}

type Clientsets interface {
	Kubernetes() kubernetes.Interface
	APIExtensions() apiextensions.Interface
	Furiko() furiko.Interface
	Argo() argo.Interface
}

type contextClientsets struct {
	kubernetes    kubernetes.Interface
	furiko        furiko.Interface
	argo          argo.Interface
	apiExtensions *apiextensions.Clientset
}

func (c *contextClientsets) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

func (c *contextClientsets) APIExtensions() apiextensions.Interface {
	return c.apiExtensions
}

func (c *contextClientsets) Furiko() furiko.Interface {
	return c.furiko
}

func (c *contextClientsets) Argo() argo.Interface {
	return c.argo
}

func SetUpClientsets(cfg *rest.Config) (Clientsets, error) {
	var err error
	clientsets := &contextClientsets{}

	clientsets.kubernetes, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create kubernetes client")
	}
	clientsets.apiExtensions, err = apiextensions.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create apiextensions client")
	}
	clientsets.furiko, err = furiko.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create furiko client")
	}
	clientsets.argo, err = argo.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create argo client")
	}

	return clientsets, nil
}
