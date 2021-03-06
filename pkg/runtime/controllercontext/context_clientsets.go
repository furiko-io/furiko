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
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	furiko "github.com/furiko-io/furiko/pkg/generated/clientset/versioned"
)

func (c *ctrlContext) Clientsets() Clientsets {
	return c.clientsets
}

type Clientsets interface {
	Kubernetes() kubernetes.Interface
	Furiko() furiko.Interface
}

type contextClientsets struct {
	kubernetes kubernetes.Interface
	furiko     furiko.Interface
}

func (c *contextClientsets) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

func (c *contextClientsets) Furiko() furiko.Interface {
	return c.furiko
}

func SetUpClientsets(cfg *rest.Config) (Clientsets, error) {
	var err error
	clientsets := &contextClientsets{}

	clientsets.kubernetes, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create kubernetes client")
	}
	clientsets.furiko, err = furiko.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create furiko client")
	}

	return clientsets, nil
}
