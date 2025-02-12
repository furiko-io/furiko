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

package httphandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
)

// ServeWebhooks prepares admission webhooks to be served.
// TODO(irvinlim): Handle conversion webhooks here in the future too.
func ServeWebhooks(mux *http.ServeMux, webhooks []controllermanager.Webhook) {
	for _, webhook := range webhooks {
		mux.HandleFunc(webhook.Path(), HandleAdmissionWebhook(webhook))
		klog.V(4).Infof("httphandler: added http handler for webhook at %v", webhook.Path())
	}
}

// HandleAdmissionWebhook returns a http.HandlerFunc for a Webhook.
func HandleAdmissionWebhook(webhook controllermanager.Webhook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		statusCode := http.StatusOK

		resp, err := handleAdmissionReview(webhook, r)

		// Handle errors as admission failures.
		if err != nil {
			status := kerrors.NewInternalError(err).Status()
			resp = &admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					Result: &status,
				},
			}
			statusCode = http.StatusInternalServerError
			if status.Code > 0 {
				statusCode = int(status.Code)
			}
		}

		handleWebhookResponse(webhook, w, statusCode, resp, startTime)
	}
}

// handleAdmissionReview is the entrypoint to handle an incoming AdmissionReview request.
func handleAdmissionReview(webhook controllermanager.Webhook, r *http.Request) (*admissionv1.AdmissionReview, error) {
	// Handle incoming request.
	var req admissionv1.AdmissionReview
	if err := handleWebhookRequest(webhook, r, &req); err != nil {
		return nil, err
	}

	// Ensure request body has a Request.
	if req.Request == nil {
		err := errors.New("missing request")
		klog.ErrorS(err, "httphandler: missing request in AdmissionReview",
			"name", webhook.Name(),
			"path", webhook.Path(),
		)
		return nil, err
	}

	// Delegate to webhook handlers.
	resp, err := handleWebhookDelegate(r.Context(), webhook, req)
	if err != nil {
		klog.ErrorS(err, "httphandler: webhook handler error",
			"name", webhook.Name(),
			"path", webhook.Path(),
		)
		return nil, err
	}

	return resp, nil
}

func handleWebhookRequest(
	webhook controllermanager.Webhook,
	r *http.Request,
	req interface{},
) error {
	// Verify the content type is accurate.
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		err := fmt.Errorf(`expected application/json, got "%v"`, contentType)
		klog.ErrorS(err, "incorrect content-type",
			"name", webhook.Name(),
			"path", webhook.Path(),
		)
		return errors.Wrapf(err, "incorrect content-type")
	}

	// Prepare MultiWriter to capture request body.
	buf := new(bytes.Buffer)
	var body io.Reader = r.Body
	if klog.V(4).Enabled() {
		body = io.TeeReader(r.Body, buf)
	}

	if err := json.NewDecoder(body).Decode(req); err != nil {
		klog.ErrorS(err, "httphandler: cannot decode as AdmissionReview",
			"name", webhook.Name(),
			"path", webhook.Path(),
		)
		return errors.Wrapf(err, "cannot decode as AdmissionReview")
	}

	// Print debug logs for input request.
	if body, err := io.ReadAll(buf); err == nil {
		klog.V(4).InfoS(
			"httphandler: received request",
			"name", webhook.Name(),
			"path", webhook.Path(),
			"body", body,
			"url", r.URL.String(),
			"method", r.Method,
		)
	}

	return nil
}

func handleWebhookDelegate(
	ctx context.Context,
	webhook controllermanager.Webhook,
	req admissionv1.AdmissionReview,
) (resp *admissionv1.AdmissionReview, err error) {
	// Recover from panic in webhook handler.
	defer func() {
		if recovered := recover(); recovered != nil {
			logPanic(recovered)
			err = fmt.Errorf("panic in webhook handler: %v", recovered)
		}
	}()

	// Delegate to webhook handlers.
	webhookResp, err := webhook.Handle(ctx, req.Request)
	if err != nil {
		return nil, errors.Wrapf(err, "webhook handler error")
	}

	// Form AdmissionReview response.
	resp = &admissionv1.AdmissionReview{}
	resp.SetGroupVersionKind(req.GroupVersionKind())
	resp.Response = webhookResp
	resp.Response.UID = req.Request.UID

	return resp, nil
}

func handleWebhookResponse(
	webhook controllermanager.Webhook,
	w http.ResponseWriter,
	status int,
	resp interface{},
	startTime time.Time,
) {
	// Write status code.
	w.WriteHeader(status)

	// Prepare MultiWriter to capture response body.
	buf := new(bytes.Buffer)
	var out io.Writer = w
	if klog.V(4).Enabled() {
		out = io.MultiWriter(w, buf)
	}

	if err := json.NewEncoder(out).Encode(resp); err != nil {
		klog.ErrorS(err, "httphandler: cannot encode AdmissionResponse",
			"name", webhook.Name(),
			"path", webhook.Path(),
		)
		http.Error(w, fmt.Sprint("cannot encode AdmissionResponse:", err), http.StatusInternalServerError)
		return
	}

	// Print logs that were piped.
	if klog.V(4).Enabled() {
		klog.V(4).InfoS("httphandler: returning webhook response",
			"name", webhook.Name(),
			"path", webhook.Path(),
			"body", buf.String(),
			"elapsed", time.Since(startTime),
		)
	}
}

func logPanic(r interface{}) {
	// Adapted from kubernetes/apimachinery/util/runtime.
	const size = 64 << 10
	stacktrace := make([]byte, size)
	stacktrace = stacktrace[:runtime.Stack(stacktrace, false)]
	if _, ok := r.(string); ok {
		klog.Errorf("Observed a panic: %s\n%s", r, stacktrace)
	} else {
		klog.Errorf("Observed a panic: %#v (%v)\n%s", r, r, stacktrace)
	}
}
