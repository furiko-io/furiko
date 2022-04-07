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

package jobvalidatingwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	executionv1alpha1 "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/validation"
	executioninformers "github.com/furiko-io/furiko/pkg/generated/informers/externalversions/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
)

const (
	webhookName             = "JobValidatingWebhook"
	waitForCacheSyncTimeout = 3 * time.Minute
)

var (
	readiness uint64
)

type Webhook struct {
	controllercontext.ContextInterface
	jobconfigInformer executioninformers.JobConfigInformer
	hasSynced         []cache.InformerSynced
}

var _ controllermanager.Webhook = (*Webhook)(nil)

func NewWebhook(ctrlContext controllercontext.ContextInterface) (*Webhook, error) {
	jobconfigInformer := ctrlContext.Informers().Furiko().Execution().V1alpha1().JobConfigs()

	webhook := &Webhook{
		ContextInterface:  ctrlContext,
		jobconfigInformer: jobconfigInformer,
		hasSynced: []cache.InformerSynced{
			jobconfigInformer.Informer().HasSynced,
		},
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
	defer utilruntime.HandleCrash()
	klog.InfoS("jobvalidatingwebhook: starting webhook")

	if err := controllerutil.WaitForNamedCacheSyncWithTimeout(ctx, webhookName, waitForCacheSyncTimeout,
		w.hasSynced...); err != nil {
		klog.ErrorS(err, "jobvalidatingwebhook: cache sync timeout")
		return err
	}

	atomic.StoreUint64(&readiness, 1)
	klog.InfoS("jobvalidatingwebhook: started webhook")
	return nil
}

func (w *Webhook) Shutdown(_ context.Context) {
	klog.InfoS("jobvalidatingwebhook: stopped webhook")
}

func (w *Webhook) Path() string {
	return fmt.Sprintf("/validating/jobs.%v", executiongroup.GroupName)
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
	if gvk != executionv1alpha1.GVKJob {
		return nil, fmt.Errorf("unhandled GroupVersionKind: %v", gvk)
	}

	var oldRj *executionv1alpha1.Job
	rj := &executionv1alpha1.Job{}

	switch req.Operation {
	case admissionv1.Update:
		oldRj = &executionv1alpha1.Job{}
		if err := json.Unmarshal(req.OldObject.Raw, oldRj); err != nil {
			return nil, errors.Wrapf(err, "cannot decode object as executionv1alpha1.Job")
		}
		fallthrough

	case admissionv1.Create:
		if err := json.Unmarshal(req.Object.Raw, rj); err != nil {
			return nil, errors.Wrapf(err, "cannot decode object as executionv1alpha1.Job")
		}

	default:
		klog.InfoS("jobvalidatingwebhook: unhandled operation", "operation", req.Operation)
		resp.Allowed = true
		return resp, nil
	}

	// Validate the Job.
	if errs := w.Validate(req, oldRj, rj); len(errs) > 0 {
		status := kerrors.NewInvalid(gvk.GroupKind(), rj.GetName(), errs).Status()
		resp.Result = &status
	} else {
		resp.Allowed = true
	}

	return resp, nil
}

// Validate the incoming admission request for a JobConfig and return an
// ErrorList of aggregated errors.
func (w *Webhook) Validate(req *admissionv1.AdmissionRequest, oldRj, rj *executionv1alpha1.Job) field.ErrorList {
	validator := validation.NewValidator(w)

	errorList := validator.ValidateJob(rj)
	switch req.Operation {
	case admissionv1.Create:
		errorList = append(errorList, validator.ValidateJobCreate(rj)...)
	case admissionv1.Update:
		errorList = append(errorList, validator.ValidateJobUpdate(oldRj, rj)...)
	}

	return errorList
}
