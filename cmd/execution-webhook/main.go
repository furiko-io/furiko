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

package main

import (
	"context"
	"errors"
	"flag"

	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/webhooks/jobconfigmutatingwebhook"
	"github.com/furiko-io/furiko/pkg/execution/webhooks/jobconfigvalidatingwebhook"
	"github.com/furiko-io/furiko/pkg/execution/webhooks/jobmutatingwebhook"
	"github.com/furiko-io/furiko/pkg/execution/webhooks/jobvalidatingwebhook"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/httphandler"
	"github.com/furiko-io/furiko/pkg/runtime/util"
	"github.com/furiko-io/furiko/pkg/utils/yaml"
)

// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobs/status,verbs=get
// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobconfigs/status,verbs=get

func main() {
	initFlags()
	klog.InitFlags(nil)
	defer klog.Flush()
	flag.Parse()

	// Read bootstrap configuration from file. This set of configuration only
	// controls options for starting up. Dynamic configuration (i.e. runtime
	// behaviour of individual controllers/webhooks) is instead controlled via
	// ConfigMap or some dynamic configuration source.
	//
	// Changes will not take effect until the server is restarted.
	var options configv1alpha1.ExecutionWebhookConfig
	if configFile != "" {
		klog.Infof("loading configuration from %v", configFile)
		if err := util.UnmarshalFromFile(configFile, &options); err != nil {
			klog.Fatalf(`cannot read config at "%v": %v`, configFile, err)
		}
	}

	// Dump loaded configuration.
	optionsMarshaled, err := yaml.MarshalToString(options)
	if err != nil {
		klog.Fatalf("cannot marshal configuration: %v", err)
	}
	klog.Infof("bootstrap configuration loaded:\n%v", optionsMarshaled)

	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		klog.Fatalf("cannot get kubeconfig: %v", err)
	}

	// Prepare controller context.
	klog.Info("setting up controllercontext")
	ctrlContext, err := controllercontext.NewForConfig(kubeconfig, &options.BootstrapConfigSpec)
	if err != nil {
		klog.Fatalf("cannot initialize controllercontext: %v", err)
	}

	// Create webhook manager.
	klog.Info("setting up webhook manager")
	mgr := controllermanager.NewWebhookManager(ctrlContext)

	ctx := ctrl.SetupSignalHandler()

	// Set up webhooks.
	webhookFactories := GetWebhookFactories()
	webhooks := make([]controllermanager.Webhook, 0, len(webhookFactories))
	for _, factory := range webhookFactories {
		webhook, err := factory.New(ctrlContext)
		if err != nil {
			klog.Fatalf("cannot initialize webhook %v: %v", factory.Name(), err)
		}
		webhooks = append(webhooks, webhook)
		mgr.Add(webhook)
	}

	// Start webhook server in background.
	go func() {
		if err := httphandler.ListenAndServeWebhooks(ctx, options.Webhooks, webhooks); err != nil {
			klog.Fatalf("cannot start webhooks server: %v", err)
		}
	}()

	// Start HTTP server in background.
	go func() {
		if err := httphandler.ListenAndServe(ctx, options.HTTP, mgr); err != nil {
			klog.Fatalf("cannot start http handlers: %v", err)
		}
	}()

	// Start webhook manager.
	klog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		klog.Fatalf("cannot start webhook manager: %v", err)
	}

	// Wait for signal
	<-ctx.Done()
	klog.Info("signal received, shutting down...")

	// Shutdown webhooks, and block up to a timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), teardownTimeout)
	defer cancel()
	mgr.ShutdownAndWait(shutdownCtx)
}

// WebhookFactory is able to create a new Webhook.
type WebhookFactory interface {
	Name() string
	New(ctx controllercontext.Context) (controllermanager.Webhook, error)
}

// GetWebhookFactories returns a list of WebhookFactory implementations that
// should be created by this webhook server.
func GetWebhookFactories() []WebhookFactory {
	return []WebhookFactory{
		jobconfigvalidatingwebhook.NewFactory(),
		jobconfigmutatingwebhook.NewFactory(),
		jobvalidatingwebhook.NewFactory(),
		jobmutatingwebhook.NewFactory(),
	}
}
