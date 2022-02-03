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

package controllermanager

import (
	"context"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/klog/v2"

	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

type Webhook interface {
	Runnable

	// Name of the webhook.
	Name() string

	// Start blocks until the webhook is ready. Once the method returns, it is
	// assumed to be Ready. If the context is canceled, Start should return timely
	// with the context error.
	Start(ctx context.Context) error

	// Ready specifies if the webhook is ready to receive requests.
	Ready() bool

	// Path of the webhook.
	Path() string

	// Handle the incoming request and returns a response.
	Handle(ctx context.Context, req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error)
}

// WebhookManager takes care of the lifecycles of webhooks, and the liveness and
// readiness status of the server.
type WebhookManager struct {
	*BaseManager
	webhooks []Webhook
}

func NewWebhookManager(ctrlContext *controllercontext.Context) *WebhookManager {
	return &WebhookManager{
		BaseManager: NewBaseManager(ctrlContext),
	}
}

// Add a new runnable to the webhook manager to be managed. Will not take
// effect once Start is called.
func (m *WebhookManager) Add(runnables ...Runnable) {
	m.BaseManager.Add(runnables...)

	for _, runnable := range runnables {
		if c, ok := runnable.(Webhook); ok {
			m.webhooks = append(m.webhooks, c)
		}
	}
}

// Start will start up all webhooks and returns.
//
// If there is an error in starting, this method returns an error. When the
// context is canceled, all webhooks should stop as well.
func (m *WebhookManager) Start(ctx context.Context) error {
	klog.Infof("controllermanager: starting webhook manager")

	// Start the base manager.
	if err := m.BaseManager.Start(ctx); err != nil {
		return err
	}

	// Start webhooks and wait until they are ready. The webhook server cannot serve
	// requests until they are fully started, and we have to ensure that they are
	// ready before passing the readiness probe.
	klog.Infof("controllermanager: starting webhooks")
	if err := StartWebhooksAndWait(ctx, m.webhooks); err != nil {
		return errors.Wrapf(err, "cannot start webhooks")
	}
	klog.Infof("controllermanager: webhooks started and ready")

	// Webhook manager is now ready and started.
	m.makeReady()
	m.makeStarted()

	return nil
}

// StartWebhooksAndWait starts all the given Webhooks in the background and
// blocks until they are all ready. If any controller's Start method returns an
// error, this method returns and cancels the context. To terminate controllers,
// cancel the passed context.
func StartWebhooksAndWait(ctx context.Context, webhooks []Webhook) error {
	grp, ctx := errgroup.WithContext(ctx)

	for _, webhook := range webhooks {
		webhook := webhook
		grp.Go(func() error {
			return webhook.Start(ctx)
		})
	}

	return grp.Wait()
}
