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

package jobconfigmutatingwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	executionv1alpha1 "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/mutation"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/utils/cmp"
	"github.com/furiko-io/furiko/pkg/utils/webhook"
)

const (
	webhookName = "JobConfigMutatingWebhook"
)

var (
	readiness uint64 = 0
)

type Webhook struct {
	controllercontext.ContextInterface
}

var _ controllermanager.Webhook = (*Webhook)(nil)

func NewWebhook(ctrlContext controllercontext.ContextInterface) (*Webhook, error) {
	webhook := &Webhook{
		ContextInterface: ctrlContext,
	}
	return webhook, nil
}

func (w *Webhook) Name() string {
	return webhookName
}

func (w *Webhook) Ready() bool {
	return readiness == 1
}

func (w *Webhook) Start(ctx context.Context) error {
	atomic.StoreUint64(&readiness, 1)
	klog.InfoS("jobconfigmutatingwebhook: started webhook")
	return nil
}

func (w *Webhook) Shutdown(ctx context.Context) {
	klog.InfoS("jobconfigmutatingwebhook: stopped webhook")
}

func (w *Webhook) Path() string {
	return fmt.Sprintf("/mutating/jobconfigs.%v", executiongroup.GroupName)
}

func (w *Webhook) Handle(
	_ context.Context,
	req *admissionv1.AdmissionRequest,
) (*admissionv1.AdmissionResponse, error) {
	resp := &admissionv1.AdmissionResponse{}
	gvk := schema.GroupVersionKind{
		Group:   req.Kind.Group,
		Version: req.Kind.Version,
		Kind:    req.Kind.Kind,
	}

	// Only handle specific GVK.
	// TODO(irvinlim): Handle different versions separately.
	if gvk != executionv1alpha1.GVKJobConfig {
		return nil, fmt.Errorf("unhandled GroupVersionKind: %v", gvk)
	}

	rjc := &executionv1alpha1.JobConfig{}
	var existing *executionv1alpha1.JobConfig

	switch req.Operation {
	case admissionv1.Update:
		existing = &executionv1alpha1.JobConfig{}
		if err := json.Unmarshal(req.OldObject.Raw, existing); err != nil {
			return nil, errors.Wrapf(err, "cannot decode object as executionv1alpha1.JobConfig")
		}
		fallthrough

	case admissionv1.Create:
		if err := json.Unmarshal(req.Object.Raw, rjc); err != nil {
			return nil, errors.Wrapf(err, "cannot decode object as executionv1alpha1.JobConfig")
		}

	default:
		klog.InfoS("jobconfigmutatingwebhook: unhandled operation", "operation", req.Operation)
		resp.Allowed = true
		return resp, nil
	}

	// Patch the JobConfig.
	newRjc := rjc.DeepCopy()
	result := w.Patch(req, existing, newRjc)

	// Encountered validation error.
	if len(result.Errors) > 0 {
		status := kerrors.NewInvalid(gvk.GroupKind(), rjc.GetName(), result.Errors).Status()
		resp.Result = &status
		return resp, nil
	}

	// Patch succeeded, we will allow let it through
	resp.Allowed = true
	resp.Warnings = result.Warnings

	isEqual, err := cmp.IsJSONEqual(rjc, newRjc)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot compare objects")
	}

	// Create patch if not equal.
	if !isEqual {
		patch, err := cmp.CreateJSONPatch(rjc, newRjc)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create jsonpatch")
		}

		patchType := admissionv1.PatchTypeJSONPatch
		resp.Patch = patch
		resp.PatchType = &patchType
	}

	return resp, nil
}

func (w *Webhook) Patch(req *admissionv1.AdmissionRequest, oldRjc, rjc *executionv1alpha1.JobConfig) *webhook.Result {
	mutator := mutation.NewMutator(w)
	result := mutator.MutateJobConfig(rjc)

	switch req.Operation {
	case admissionv1.Create:
		result.Merge(mutator.MutateCreateJobConfig(rjc))
	case admissionv1.Update:
		result.Merge(mutator.MutateUpdateJobConfig(oldRjc, rjc))
	}

	return result
}
