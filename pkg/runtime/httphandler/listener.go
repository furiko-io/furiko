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
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
)

var (
	defaultHTTPConfig = &configv1alpha1.HTTPSpec{
		BindAddress: ":8080",
	}

	defaultWebhooksConfig = &configv1alpha1.WebhookServerSpec{
		BindAddress: ":9443",
	}
)

// ListenAndServeHTTP listens on the given TCP address and gracefully stops when the
// given context is canceled, setting up all HTTP handlers.
func ListenAndServeHTTP(
	ctx context.Context,
	config *configv1alpha1.HTTPSpec,
	healthHandler HealthHandler,
	bootstrapConfig runtime.Object,
	configsGetter controllercontext.Configs,
) error {
	if config == nil {
		config = defaultHTTPConfig
	}
	addr := config.BindAddress
	if addr == "" {
		addr = defaultHTTPConfig.BindAddress
	}

	server := NewServer(addr)
	server.RegisterCommonHandlers(
		config,
		metrics.Registry,
		healthHandler,
		&defaultDebugHandler{
			bootstrapConfig: bootstrapConfig,
			configsGetter:   configsGetter,
		},
	)
	return server.ListenAndServe(ctx)
}

// ListenAndServeWebhooks listens on the given TCP address and gracefully stops when the
// given context is canceled, setting up all webhooks handlers.
func ListenAndServeWebhooks(
	ctx context.Context,
	config *configv1alpha1.WebhookServerSpec,
	webhooks []controllermanager.Webhook,
) error {
	if config == nil {
		config = defaultWebhooksConfig
	}
	addr := config.BindAddress
	if addr == "" {
		addr = defaultWebhooksConfig.BindAddress
	}

	mux := http.NewServeMux()
	server := newTLSServer(&http.Server{
		Addr:    addr,
		Handler: mux,
	}, config.TLSCertFile, config.TLSPrivateKeyFile)

	ServeWebhooks(mux, webhooks)
	return listenAndServe(ctx, addr, server)
}

// ListeningServer is implemented by both http.Server (HTTP) and tlsServer (HTTPS).
type ListeningServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

var _ ListeningServer = (*http.Server)(nil)

// listenAndServe binds to the given address, listening for connections to be served by the given ListeningServer.
func listenAndServe(ctx context.Context, addr string, server ListeningServer) error {
	go func() {
		<-ctx.Done()
		klog.Infof("httphandler: shutting down http server on %v", addr)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			klog.ErrorS(err, "httphandler: error while shutting down", "addr", addr)
		}
		klog.Infof("httphandler: http server shut down on %v", addr)
	}()

	klog.Infof("httphandler: http server listening on %v", addr)
	if err := server.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}
