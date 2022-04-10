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
	"flag"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/bootstrap"
	"github.com/furiko-io/furiko/pkg/execution/controllers/croncontroller"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobconfigcontroller"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobcontroller"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobqueuecontroller"
	"github.com/furiko-io/furiko/pkg/execution/stores/activejobstore"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
	"github.com/furiko-io/furiko/pkg/runtime/httphandler"
)

// +kubebuilder:rbac:groups="",resources=events;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=execution.furiko.io,resources=jobconfigs/finalizers,verbs=update

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
	var options configv1alpha1.ExecutionControllerConfig
	if configFile != "" {
		klog.Infof("loading configuration from %v", configFile)
		if err := bootstrap.UnmarshalFromFile(configFile, &options); err != nil {
			klog.Fatalf(`cannot read config at "%v": %v`, configFile, err)
		}
	}

	// Dump loaded configuration.
	optionsMarshaled, err := bootstrap.Marshal(options)
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

	// Create controller manager.
	klog.Info("setting up controller manager")
	mgr, err := controllermanager.NewManager(
		ctrlContext,
		options.ControllerManagerConfigSpec,
		"execution-controller",
	)
	if err != nil {
		klog.Fatalf("cannot initialize controller manager: %v", err)
	}

	// Set up stores.
	for _, factory := range GetStoreFactories() {
		klog.Infof("setting up store %v", factory.Name())
		store, err := factory.New(ctrlContext)
		if err != nil {
			klog.Fatalf("cannot initialize store %v: %v", factory.Name(), err)
		}
		mgr.AddStore(store)
		ctrlContext.Stores().Register(store)
	}

	// Set up controllers.
	for _, factory := range GetControllerFactories() {
		concurrencySpec := options.ControllerConcurrency
		if concurrencySpec == nil {
			concurrencySpec = &configv1alpha1.ExecutionControllerConcurrencySpec{}
		}
		klog.Infof("setting up controller %v", factory.Name())
		controller, err := factory.New(ctrlContext, concurrencySpec)
		if err != nil {
			klog.Fatalf("cannot initialize controller %v: %v", factory.Name(), err)
		}
		mgr.Add(controller)
	}

	ctx := ctrl.SetupSignalHandler()

	// Start HTTP server in background.
	go func() {
		if err := httphandler.ListenAndServe(ctx, options.HTTP, mgr); err != nil {
			klog.Fatalf("cannot start http handlers: %v", err)
		}
	}()

	klog.Info("starting manager")
	if err := mgr.Start(ctx, startupTimeout); err != nil && !errors.Is(err, context.Canceled) {
		klog.Fatalf("cannot start controller manager: %v", err)
	}
	klog.Info("all controllers started")

	// Wait for signal
	<-ctx.Done()
	klog.Info("signal received, shutting down...")

	// Shutdown controller manager, and block up to a timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), teardownTimeout)
	defer cancel()
	mgr.ShutdownAndWait(shutdownCtx)
}

// ControllerFactory is able to create a new Controller.
type ControllerFactory interface {
	Name() string
	New(ctx controllercontext.Context,
		concurrency *configv1alpha1.ExecutionControllerConcurrencySpec) (controllermanager.Controller, error)
}

// GetControllerFactories returns a list of ControllerFactory implementations
// that should be created by this controller manager.
func GetControllerFactories() []ControllerFactory {
	return []ControllerFactory{
		croncontroller.NewFactory(),
		jobcontroller.NewFactory(),
		jobconfigcontroller.NewFactory(),
		jobqueuecontroller.NewFactory(),
	}
}

type StoreFactory interface {
	Name() string
	New(ctx controllercontext.Context) (controllermanager.Store, error)
}

// GetStoreFactories returns a list of StoreFactory implementations
// that should be created by this controller manager.
func GetStoreFactories() []StoreFactory {
	return []StoreFactory{
		activejobstore.NewFactory(),
	}
}
