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

package common

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kr/text"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/utils/jsonyaml"
)

const (
	DefaultErrorExitCode = 1
)

var (
	ctrlContext controllercontext.Context
)

type RunEFunc func(cmd *cobra.Command, args []string) error

// RunAllE composes multiple RunEFunc together.
func RunAllE(funcs ...RunEFunc) RunEFunc {
	return func(cmd *cobra.Command, args []string) error {
		for _, f := range funcs {
			if err := f(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}

// NewContext returns a common context from the cobra command.
func NewContext(cmd *cobra.Command) (controllercontext.Context, error) {
	kubeconfig, err := GetKubeConfig(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get kubeconfig")
	}
	return controllercontext.NewForConfig(kubeconfig, &configv1alpha1.BootstrapConfigSpec{})
}

// GetKubeConfig returns the desired kubeconfig.
func GetKubeConfig(cmd *cobra.Command) (*rest.Config, error) {
	// Get the --context and --cluster flags.
	kubeconfigContext := GetFlagString(cmd, "context")
	kubeconfigCluster := GetFlagString(cmd, "cluster")

	// Read the --kubeconfig flag.
	if kubeconfig := GetFlagString(cmd, "kubeconfig"); kubeconfig != "" {
		klog.V(1).InfoS("loading kubeconfig from --kubeconfig",
			"path", kubeconfig,
		)

		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
			&clientcmd.ConfigOverrides{
				CurrentContext: kubeconfigContext,
				Context: api.Context{
					Cluster: kubeconfigCluster,
				},
			},
		).ClientConfig()
	}

	// If the recommended kubeconfig env variable is not specified,
	// try the in-cluster config.
	kubeconfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if len(kubeconfigPath) == 0 {
		if c, err := rest.InClusterConfig(); err == nil {
			klog.V(1).InfoS("successfully loaded in-cluster kubeconfig")
			return c, nil
		}
	}

	// If the recommended kubeconfig env variable is set, or there
	// is no in-cluster config, try the default recommended locations.
	//
	// NOTE: For default config file locations, upstream only checks
	// $HOME for the user's home directory, but we can also try
	// os/user.HomeDir when $HOME is unset.
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if _, ok := os.LookupEnv("HOME"); !ok {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("could not get current user: %v", err)
		}
		loadingRules.Precedence = append(loadingRules.Precedence, filepath.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName))
	}

	klog.V(1).InfoS("loading default kubeconfig from recommended locations",
		"pathPrecedence", strings.Join(loadingRules.Precedence, ":"),
	)

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{
			CurrentContext: kubeconfigContext,
			Context: api.Context{
				Cluster: kubeconfigCluster,
			},
		},
	).ClientConfig()
}

// PrerunWithKubeconfig is a pre-run function that will set up the common context when kubeconfig is needed.
func PrerunWithKubeconfig(cmd *cobra.Command, _ []string) error {
	return SetupCtrlContext(cmd)
}

// SetupCtrlContext sets up the common context.
// TODO(irvinlim): We currently reuse controllercontext, but most of it is unusable for CLI interfaces.
// We should create a new common context as needed.
func SetupCtrlContext(cmd *cobra.Command) error {
	// Already set up previously.
	if ctrlContext != nil {
		return nil
	}

	newContext, err := NewContext(cmd)
	if err != nil {
		return err
	}
	ctrlContext = newContext
	return nil
}

// GetCtrlContext retrieves the common context.
func GetCtrlContext() controllercontext.Context {
	return ctrlContext
}

// SetCtrlContext explicitly sets the common context.
func SetCtrlContext(cc controllercontext.Context) {
	ctrlContext = cc
}

// GetNamespace returns the namespace to use depending on what was defined in the flags.
func GetNamespace(cmd *cobra.Command) (string, error) {
	// If --all-namespaces is defined on the command, use it first.
	if allNamespaces, ok := GetFlagBoolIfExists(cmd, "all-namespaces"); ok && allNamespaces {
		return metav1.NamespaceAll, nil
	}

	// Read the --namespace flag if specified.
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return "", errors.Wrapf(err, "cannot get value of --namespace")
	}

	// Otherwise, fall back to the default namespace.
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	return namespace, nil
}

// GetDynamicConfig loads the dynamic config by name and unmarshals to out.
// TODO(irvinlim): If the current user does not have permissions to read the
// ConfigMap, or the ConfigMap uses a different name/namespace, we should
// gracefully handle this case.
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
func GetOutputFormat(cmd *cobra.Command) printer.OutputFormat {
	v := GetFlagString(cmd, "output")
	output := printer.OutputFormat(v)
	klog.V(4).InfoS("using output format", "format", output)
	return output
}

// GetFlagBool gets the boolean value of a flag.
func GetFlagBool(cmd *cobra.Command, flag string) bool {
	b, err := cmd.Flags().GetBool(flag)
	if err != nil {
		klog.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}
	return b
}

// GetFlagBoolIfExists gets the boolean value of a flag if it exists.
func GetFlagBoolIfExists(cmd *cobra.Command, flag string) (val bool, ok bool) {
	if cmd.Flags().Lookup(flag) != nil {
		val := GetFlagBool(cmd, flag)
		return val, true
	}
	return false, false
}

// GetFlagString gets the string value of a flag.
func GetFlagString(cmd *cobra.Command, flag string) string {
	v, err := cmd.Flags().GetString(flag)
	if err != nil {
		klog.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}
	return v
}

// GetFlagInt64 gets the int64 value of a flag.
func GetFlagInt64(cmd *cobra.Command, flag string) int64 {
	v, err := cmd.Flags().GetInt64(flag)
	if err != nil {
		klog.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}
	return v
}

// PrepareExample replaces the root command name and indents all lines.
func PrepareExample(example string) string {
	example = strings.TrimPrefix(example, "\n")
	example = strings.ReplaceAll(example, "{{.CommandName}}", CommandName)
	return text.Indent(example, "  ")
}

// Fatal terminates the command immediately and writes to stderr.
func Fatal(err error, code int) {
	msg := err.Error()
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		_, _ = fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(code)
}
