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

package cmd

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/utils/jsonyaml"
)

var (
	ctrlContext controllercontext.Context
)

// Runner knows how to run a command.
type Runner interface {
	Run(ctx context.Context, cmd *cobra.Command, args []string) error
}

// ToRunE converts a Runner with a context to a RunE func.
func ToRunE(ctx context.Context, executor Runner) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return executor.Run(ctx, cmd, args)
	}
}

// NewContext returns a common context from the cobra command.
func NewContext(_ *cobra.Command) (controllercontext.Context, error) {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get kubeconfig")
	}
	return controllercontext.NewForConfig(kubeconfig, &configv1alpha1.BootstrapConfigSpec{})
}

// PrerunWithKubeconfig is a pre-run function that will set up the common context when kubeconfig is needed.
// TODO(irvinlim): We currently reuse controllercontext, but most of it is unusable for CLI interfaces.
//  We should create a new common context as needed.
func PrerunWithKubeconfig(cmd *cobra.Command, _ []string) error {
	newContext, err := NewContext(cmd)
	if err != nil {
		return err
	}
	ctrlContext = newContext
	return nil
}

// GetNamespace returns the namespace to use depending on what was defined in the flags.
func GetNamespace(cmd *cobra.Command) (string, error) {
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return "", err
	}
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	return namespace, nil
}

// GetDynamicConfig loads the dynamic config by name and unmarshals to out.
// TODO(irvinlim): If the current user does not have permissions to read the
//  ConfigMap, or the ConfigMap uses a different name/namespace, we should
//  gracefully handle this case.
func GetDynamicConfig(ctx context.Context, cmd *cobra.Command, name configv1alpha1.ConfigName, out interface{}) error {
	cfgNamespace, err := cmd.Flags().GetString("dynamic-config-namespace")
	if err != nil {
		return err
	}
	cfgName, err := cmd.Flags().GetString("dynamic-config-name")
	if err != nil {
		return err
	}

	klog.V(2).InfoS("fetching dynamic config", "namespace", cfgNamespace, "name", cfgName)
	cm, err := ctrlContext.Clientsets().Kubernetes().CoreV1().ConfigMaps(cfgNamespace).
		Get(ctx, cfgName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot load dynamic config")
	}

	data := cm.Data[string(name)]
	klog.V(2).InfoS("fetched dynamic config", "data", data)

	return jsonyaml.UnmarshalString(data, out)
}

// GetOutputFormat returns the output format as parsed by the flag.
func GetOutputFormat(cmd *cobra.Command) (OutputFormat, error) {
	v, err := cmd.Flags().GetString("output")
	if err != nil {
		return "", err
	}
	output := OutputFormat(v)
	klog.V(4).InfoS("using output format", "format", output)
	return output, nil
}
