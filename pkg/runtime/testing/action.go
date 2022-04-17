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

package testing

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

var (
	defaultVerbs = []string{
		"create",
		"update",
		"patch",
		"delete",
	}

	resourcePod       = NewGroupVersionResource("core", "v1", "pods")
	resourceJob       = NewGroupVersionResource(executiongroup.GroupName, execution.Version, "jobs")
	resourceJobConfig = NewGroupVersionResource(executiongroup.GroupName, execution.Version, "jobconfigs")
)

// Action describes a single expected action to be taken.
type Action struct {
	ktesting.Action

	// If true, will not check the given object for equality.
	IgnoreObject bool
}

// ActionTest describes a single test to compare Actions that were recorded
// versus what is expected.
type ActionTest struct {
	// Types contains the list of verbs that should be checked.
	// Defaults to write-only verbs.
	Verbs []string

	// Actions contains a list of actions that should exist in the result.
	// It is expected that they will be in the correct order.
	Actions []Action

	// ActionGenerators is like Actions except that generating an Action may fail
	// with an error. If specified, will take precedence over Actions.
	ActionGenerators []ActionGenerator
}

// ActionGenerator generates an Action or throws an error.
type ActionGenerator func() (Action, error)

func (t ActionTest) GetVerbs() []string {
	if len(t.Verbs) == 0 {
		return defaultVerbs
	}
	return t.Verbs
}

func (t ActionTest) GetActions() ([]Action, error) {
	if len(t.ActionGenerators) > 0 {
		actions := make([]Action, 0, len(t.ActionGenerators))
		for _, gen := range t.ActionGenerators {
			action, err := gen()
			if err != nil {
				return nil, err
			}
			actions = append(actions, action)
		}
		return actions, nil
	}

	return t.Actions, nil
}

func WrapAction(action ktesting.Action) Action {
	return Action{Action: action}
}

func NewCreateAction(resource schema.GroupVersionResource, namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewCreateAction(resource, namespace, object))
}

func NewUpdateAction(resource schema.GroupVersionResource, namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewUpdateAction(resource, namespace, object))
}

func NewUpdateStatusAction(resource schema.GroupVersionResource, namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewUpdateSubresourceAction(resource, "status", namespace, object))
}

func NewPatchAction(
	resource schema.GroupVersionResource,
	namespace, name string,
	pt types.PatchType,
	patch []byte,
) Action {
	return WrapAction(ktesting.NewPatchAction(resource, namespace, name, pt, patch))
}

func NewDeleteAction(resource schema.GroupVersionResource, namespace, name string) Action {
	return WrapAction(ktesting.NewDeleteAction(resource, namespace, name))
}

func NewCreatePodAction(namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewCreateAction(resourcePod, namespace, object))
}

func NewUpdatePodAction(namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewUpdateAction(resourcePod, namespace, object))
}

func NewUpdatePodStatusAction(namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewUpdateSubresourceAction(resourcePod, "status", namespace, object))
}

func NewPatchPodAction(namespace, name string, pt types.PatchType, patch []byte) Action {
	return WrapAction(ktesting.NewPatchAction(resourcePod, namespace, name, pt, patch))
}

func NewDeletePodAction(namespace, name string) Action {
	return WrapAction(ktesting.NewDeleteAction(resourcePod, namespace, name))
}

func NewCreateJobAction(namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewCreateAction(resourceJob, namespace, object))
}

func NewUpdateJobAction(namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewUpdateAction(resourceJob, namespace, object))
}

func NewUpdateJobStatusAction(namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewUpdateSubresourceAction(resourceJob, "status", namespace, object))
}

func NewPatchJobAction(namespace, name string, pt types.PatchType, patch []byte) Action {
	return WrapAction(ktesting.NewPatchAction(resourceJob, namespace, name, pt, patch))
}

func NewDeleteJobAction(namespace, name string) Action {
	return WrapAction(ktesting.NewDeleteAction(resourceJob, namespace, name))
}

func NewCreateJobConfigAction(namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewCreateAction(resourceJobConfig, namespace, object))
}

func NewUpdateJobConfigAction(namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewUpdateAction(resourceJobConfig, namespace, object))
}

func NewUpdateJobConfigStatusAction(namespace string, object runtime.Object) Action {
	return WrapAction(ktesting.NewUpdateSubresourceAction(resourceJobConfig, "status", namespace, object))
}

func NewPatchJobConfigAction(namespace, name string, pt types.PatchType, patch []byte) Action {
	return WrapAction(ktesting.NewPatchAction(resourceJobConfig, namespace, name, pt, patch))
}

func NewDeleteJobConfigAction(namespace, name string) Action {
	return WrapAction(ktesting.NewDeleteAction(resourceJobConfig, namespace, name))
}
