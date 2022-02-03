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

package jobconfigvalidatingwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	executionv1alpha1 "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/validation"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
)

const (
	webhookName = "JobConfigValidatingWebhook"
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
	klog.InfoS("jobconfigvalidatingwebhook: started webhook")
	return nil
}

func (w *Webhook) Shutdown(_ context.Context) {
	klog.InfoS("jobconfigvalidatingwebhook: stopped webhook")
}

func (w *Webhook) Path() string {
	return fmt.Sprintf("/validating/jobconfigs.%v", executiongroup.GroupName)
}

func (w *Webhook) Handle(
	ctx context.Context,
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

	var oldRjc *executionv1alpha1.JobConfig
	rjc := &executionv1alpha1.JobConfig{}

	switch req.Operation {
	case admissionv1.Update:
		oldRjc = &executionv1alpha1.JobConfig{}
		if err := json.Unmarshal(req.OldObject.Raw, oldRjc); err != nil {
			return nil, errors.Wrapf(err, "cannot decode object as executionv1alpha1.JobConfig")
		}
		fallthrough

	case admissionv1.Create:
		if err := json.Unmarshal(req.Object.Raw, rjc); err != nil {
			return nil, errors.Wrapf(err, "cannot decode object as executionv1alpha1.JobConfig")
		}

	default:
		klog.InfoS("jobconfigvalidatingwebhook: unhandled operation", "operation", req.Operation)
		resp.Allowed = true
		return resp, nil
	}

	// Validate the JobConfig.
	if errs := w.Validate(req, oldRjc, rjc); len(errs) > 0 {
		status := kerrors.NewInvalid(gvk.GroupKind(), rjc.GetName(), errs).Status()
		resp.Result = &status
	} else {
		resp.Allowed = true
	}

	return resp, nil
}

// Validate the incoming admission request for a JobConfig and return an
// ErrorList of aggregated errors.
// nolint:lll
func (w *Webhook) Validate(req *admissionv1.AdmissionRequest, oldRjc, rjc *executionv1alpha1.JobConfig) field.ErrorList {
	validator := validation.NewValidator(w)
	errorList := validator.ValidateJobConfig(rjc)
	switch req.Operation {
	case admissionv1.Create:
		errorList = append(errorList, validator.ValidateJobConfigCreate(rjc)...)
	case admissionv1.Update:
		errorList = append(errorList, validator.ValidateJobConfigUpdate(oldRjc, rjc)...)
	}
	return errorList
}
