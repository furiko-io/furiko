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

package eventhandler

import (
	workflowv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

// WorkflowV1alpha1Workflow casts obj into *workflowv1alpha1.Workflow.
func WorkflowV1alpha1Workflow(obj interface{}) (*workflowv1alpha1.Workflow, error) {
	switch t := obj.(type) {
	case *workflowv1alpha1.Workflow:
		return t, nil
	case cache.DeletedFinalStateUnknown:
		obj, ok := t.Obj.(*workflowv1alpha1.Workflow)
		if ok {
			return obj, nil
		}
	}

	return nil, NewUnexpectedTypeError(&workflowv1alpha1.Workflow{}, obj)
}
