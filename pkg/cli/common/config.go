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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
	"github.com/furiko-io/furiko/pkg/utils/jsonyaml"
)

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

// GetCronDynamicConfig returns the cron dynamic config.
func GetCronDynamicConfig(cmd *cobra.Command) *configv1alpha1.CronExecutionConfig {
	ctx := cmd.Context()
	newCfg := &configv1alpha1.CronExecutionConfig{}
	cfgName := configv1alpha1.CronExecutionConfigName
	if err := GetDynamicConfig(ctx, cmd, cfgName, newCfg); err != nil {
		klog.ErrorS(err, "cannot fetch dynamic config, falling back to default", "name", cfgName)
		return config.DefaultCronExecutionConfig.DeepCopy()
	}
	return newCfg
}
