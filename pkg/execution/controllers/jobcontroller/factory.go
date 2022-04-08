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

package jobcontroller

import (
	v1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Name() string {
	return controllerName
}

func (f *Factory) New(
	ctrlContext controllercontext.ContextInterface,
	concurrencySpec *v1.ExecutionControllerConcurrencySpec,
) (controllermanager.Controller, error) {
	return NewController(ctrlContext, concurrencySpec.Job)
}
