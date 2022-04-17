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

package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/httphandler"
)

const (
	uid       = "b8b38ff2-6332-4839-a883-bae1860c2b36"
	name      = "object"
	namespace = "test"
)

type fakeWebhook struct {
	shouldPanic bool
	returnError error
}

func (f *fakeWebhook) Shutdown(ctx context.Context) {}

func (f *fakeWebhook) Name() string {
	return "FakeWebhook"
}

func (f *fakeWebhook) Start(ctx context.Context) error {
	return nil
}

func (f *fakeWebhook) Ready() bool {
	return true
}

func (f *fakeWebhook) Path() string {
	return "/fake"
}

func (f *fakeWebhook) Handle(
	ctx context.Context,
	req *admissionv1.AdmissionRequest,
) (*admissionv1.AdmissionResponse, error) {
	if f.shouldPanic {
		panic("webhook panic")
	} else if f.returnError != nil {
		return nil, f.returnError
	}

	resp := &admissionv1.AdmissionResponse{
		Allowed: true,
	}

	return resp, nil
}

var _ controllermanager.Webhook = (*fakeWebhook)(nil)

func TestHandleAdmissionWebhook(t *testing.T) {
	tests := []struct {
		name       string
		webhook    *fakeWebhook
		req        *http.Request
		wantStatus int
		want       *admissionv1.AdmissionReview
	}{
		{
			name:    "basic webhook",
			webhook: &fakeWebhook{},
			req: makeAdmissionReviewRequest(&admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID:       uid,
					Name:      name,
					Namespace: namespace,
				},
			}),
			wantStatus: http.StatusOK,
			want: &admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					UID:     uid,
					Allowed: true,
				},
			},
		},
		{
			name:       "missing request",
			webhook:    &fakeWebhook{},
			req:        makeAdmissionReviewRequest(&admissionv1.AdmissionReview{}),
			wantStatus: http.StatusInternalServerError,
			want: &admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					Result: makeInternalError("missing request"),
				},
			},
		},
		{
			name:       "empty body",
			webhook:    &fakeWebhook{},
			req:        makeAdmissionReviewRequest(map[string]interface{}{}),
			wantStatus: http.StatusInternalServerError,
			want: &admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					Result: makeInternalError("missing request"),
				},
			},
		},
		{
			name:       "no content-type",
			webhook:    &fakeWebhook{},
			req:        httptest.NewRequest(http.MethodPost, "/fake", bytes.NewBufferString(`{}`)),
			wantStatus: http.StatusInternalServerError,
			want: &admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					Result: makeInternalError("incorrect content-type: expected application/json, got \"\""),
				},
			},
		},
		{
			name:       "cannot parse json",
			webhook:    &fakeWebhook{},
			req:        makeJSONRequest(httptest.NewRequest(http.MethodPost, "/fake", bytes.NewBufferString(`{a`))),
			wantStatus: http.StatusInternalServerError,
			want: &admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					Result: makeInternalError("cannot decode as AdmissionReview: invalid character 'a' looking " +
						"for beginning of object key string"),
				},
			},
		},
		{
			name: "webhook panic",
			webhook: &fakeWebhook{
				shouldPanic: true,
			},
			req: makeAdmissionReviewRequest(&admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID:       uid,
					Name:      name,
					Namespace: namespace,
				},
			}),
			wantStatus: http.StatusInternalServerError,
			want: &admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					Result: makeInternalError("panic in webhook handler: webhook panic"),
				},
			},
		},
		{
			name: "webhook error",
			webhook: &fakeWebhook{
				returnError: errors.New("webhook error"),
			},
			req: makeAdmissionReviewRequest(&admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID:       uid,
					Name:      name,
					Namespace: namespace,
				},
			}),
			wantStatus: http.StatusInternalServerError,
			want: &admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					Result: makeInternalError("webhook handler error: webhook error"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := httphandler.HandleAdmissionWebhook(tt.webhook)
			w := httptest.NewRecorder()
			handler(w, tt.req)
			result := w.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.wantStatus, result.StatusCode)
			resp := &admissionv1.AdmissionReview{}
			err := json.NewDecoder(result.Body).Decode(resp)
			assert.NoError(t, err)
			if !cmp.Equal(tt.want, resp) {
				t.Errorf("returned response not equal:\ndiff = %v", cmp.Diff(tt.want, resp))
			}
		})
	}
}

// makeAdmissionReviewRequest returns a standard *http.Request for a given payload.
func makeAdmissionReviewRequest(payload interface{}) *http.Request {
	return makeJSONRequest(httptest.NewRequest(http.MethodPost, "/fake", makeAdmissionReview(payload)))
}

// makeJSONRequest returns a JSON request with content-type added.
func makeJSONRequest(req *http.Request) *http.Request {
	req.Header.Add("Content-Type", "application/json")
	return req
}

// makeAdmissionReview creates a new io.Reader for an AdmissionReview request.
func makeAdmissionReview(payload interface{}) io.Reader {
	marshaled, err := json.Marshal(payload)
	if err != nil {
		panic("cannot marshal to JSON: " + err.Error())
	}
	return bytes.NewBuffer(marshaled)
}

func makeInternalError(message string) *metav1.Status {
	status := apierrors.NewInternalError(errors.New(message)).Status()
	return &status
}
