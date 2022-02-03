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
	"time"

	kubernetes "k8s.io/client-go/informers"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	furiko "github.com/furiko-io/furiko/pkg/generated/informers/externalversions"
)

const (
	// Default value for defaultResync.
	defaultDefaultResync = time.Minute * 10
)

func (c *Context) Informers() Informers {
	return c.informers
}

type Informers interface {
	Start(ctx context.Context) error
	Kubernetes() kubernetes.SharedInformerFactory
	Furiko() furiko.SharedInformerFactory
}

type contextInformers struct {
	kubernetes kubernetes.SharedInformerFactory
	furiko     furiko.SharedInformerFactory
}

var _ Informers = &contextInformers{}

func (c *contextInformers) Kubernetes() kubernetes.SharedInformerFactory {
	return c.kubernetes
}

func (c *contextInformers) Furiko() furiko.SharedInformerFactory {
	return c.furiko
}

func SetUpInformers(clientsets Clientsets, cfg *configv1.BootstrapConfigSpec) Informers {
	defaultResync := cfg.DefaultResync.Duration
	if defaultResync == 0 {
		defaultResync = defaultDefaultResync
	}

	informers := &contextInformers{}
	informers.kubernetes = kubernetes.NewSharedInformerFactory(clientsets.Kubernetes(), defaultResync)
	informers.furiko = furiko.NewSharedInformerFactory(clientsets.Furiko(), defaultResync)
	return informers
}

func (c *contextInformers) Start(ctx context.Context) error {
	c.Kubernetes().Start(ctx.Done())
	c.Furiko().Start(ctx.Done())
	return nil
}
