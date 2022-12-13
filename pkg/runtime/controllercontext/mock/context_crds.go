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
	"sync"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/apis/core"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

const (
	ArgoWorkflowCustomResourceDefinition = "workflows.argoproj.io"
)

var (
	crdResource = core.Resource("customresourcedefinition")
)

// CRDs is a mock CRD manager. By default, it will pretend that all
// CustomResourceDefinitions are installed.
type CRDs struct {
	disabled sets.String
	mu       sync.RWMutex
}

var _ controllercontext.CRDs = (*CRDs)(nil)

func NewCRDManager() *CRDs {
	return &CRDs{
		disabled: sets.NewString(),
	}
}

func (m *CRDs) Start(ctx context.Context) error {
	return nil
}

func (m *CRDs) ArgoWorkflows() (*apiextensionsv1.CustomResourceDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.disabled[ArgoWorkflowCustomResourceDefinition]; ok {
		return nil, kerrors.NewNotFound(crdResource, ArgoWorkflowCustomResourceDefinition)
	}
	return &apiextensionsv1.CustomResourceDefinition{}, nil
}

// SetArgoWorkflows will configure whether ArgoWorkflows is enabled or not.
func (m *CRDs) SetArgoWorkflows(enabled bool) {
	m.set(enabled, ArgoWorkflowCustomResourceDefinition)
}

func (m *CRDs) set(enabled bool, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if enabled {
		m.disabled.Delete(name)
	} else {
		m.disabled.Insert(name)
	}
}
