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

package jobmutatingwebhook

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
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	executionv1alpha1 "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/mutation"
	executioninformers "github.com/furiko-io/furiko/pkg/generated/informers/externalversions/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/controllerutil"
	"github.com/furiko-io/furiko/pkg/utils/cmp"
	"github.com/furiko-io/furiko/pkg/utils/webhook"
)

const (
	webhookName             = "JobMutatingWebhook"
	waitForCacheSyncTimeout = 3 * time.Minute
)

var (
	readiness uint64 = 0
)

type Webhook struct {
	controllercontext.ContextInterface
	jobconfigInformer executioninformers.JobConfigInformer
	hasSynced         []cache.InformerSynced
}

var _ controllermanager.Webhook = (*Webhook)(nil)

func NewWebhook(ctrlContext controllercontext.ContextInterface) (*Webhook, error) {
	jobconfigInformer := ctrlContext.Informers().Furiko().Execution().V1alpha1().JobConfigs()

	hook := &Webhook{
		ContextInterface:  ctrlContext,
		jobconfigInformer: jobconfigInformer,
		hasSynced: []cache.InformerSynced{
			jobconfigInformer.Informer().HasSynced,
		},
	}

	return hook, nil
}

func (w *Webhook) Name() string {
	return webhookName
}

func (w *Webhook) Ready() bool {
	return readiness == 1
}

func (w *Webhook) Start(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	klog.InfoS("jobmutatingwebhook: starting webhook")

	if err := controllerutil.WaitForNamedCacheSyncWithTimeout(ctx, webhookName, waitForCacheSyncTimeout,
		w.hasSynced...); err != nil {
		klog.ErrorS(err, "jobmutatingwebhook: cache sync timeout")
		return err
	}

	atomic.StoreUint64(&readiness, 1)
	klog.InfoS("jobmutatingwebhook: started webhook")
	return nil
}

func (w *Webhook) Shutdown(ctx context.Context) {
	klog.InfoS("jobmutatingwebhook: stopped webhook")
}

func (w *Webhook) Path() string {
	return fmt.Sprintf("/mutating/jobs.%v", executiongroup.GroupName)
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
	if gvk != executionv1alpha1.GVKJob {
		return nil, fmt.Errorf("unhandled GroupVersionKind: %v", gvk)
	}

	rj := &executionv1alpha1.Job{}

	switch req.Operation {
	case admissionv1.Update, admissionv1.Create:
		if err := json.Unmarshal(req.Object.Raw, rj); err != nil {
			return nil, errors.Wrapf(err, "cannot decode object as executionv1alpha1.Job")
		}

	default:
		klog.InfoS("jobmutatingwebhook: unhandled operation", "operation", req.Operation)
		resp.Allowed = true
		return resp, nil
	}

	// Patch the Job.
	newRj := rj.DeepCopy()
	result := w.Patch(req, newRj)

	// Encountered validation error.
	if len(result.Errors) > 0 {
		status := kerrors.NewInvalid(gvk.GroupKind(), rj.GetName(), result.Errors).Status()
		resp.Result = &status
		return resp, nil
	}

	// Patch succeeded, we will allow let it through
	resp.Allowed = true
	resp.Warnings = result.Warnings

	isEqual, err := cmp.IsJSONEqual(rj, newRj)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot compare objects")
	}

	// Create patch if not equal.
	if !isEqual {
		patch, err := cmp.CreateJSONPatch(rj, newRj)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create jsonpatch")
		}

		patchType := admissionv1.PatchTypeJSONPatch
		resp.Patch = patch
		resp.PatchType = &patchType
	}

	return resp, nil
}

// Patch returns the result after mutating a Job in-place.
func (w *Webhook) Patch(req *admissionv1.AdmissionRequest, rj *executionv1alpha1.Job) *webhook.Result {
	mutator := mutation.NewMutator(w)
	result := mutator.MutateJob(rj)

	switch req.Operation {
	case admissionv1.Create:
		result.Merge(mutator.MutateCreateJob(rj))
	}

	return result
}
