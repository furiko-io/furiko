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

package completion

import (
	"sort"

	"github.com/spf13/cobra"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/furiko-io/furiko/pkg/cli/common"
)

// ListKubeconfigClusterCompletionFunc knows how to list clusters from the kubeconfig.
func ListKubeconfigClusterCompletionFunc(cmd *cobra.Command) ([]string, error) {
	clientConfig, err := getClientConfig(cmd)
	if err != nil {
		return nil, err
	}
	clusters := make([]string, 0, len(clientConfig.Clusters))
	for name := range clientConfig.Clusters {
		clusters = append(clusters, name)
	}
	sort.Strings(clusters)
	return clusters, nil
}

// ListKubeconfigContextCompletionFunc knows how to list contexts from the kubeconfig.
func ListKubeconfigContextCompletionFunc(cmd *cobra.Command) ([]string, error) {
	clientConfig, err := getClientConfig(cmd)
	if err != nil {
		return nil, err
	}
	contexts := make([]string, 0, len(clientConfig.Contexts))
	for name := range clientConfig.Contexts {
		contexts = append(contexts, name)
	}
	sort.Strings(contexts)
	return contexts, nil
}

func getClientConfig(cmd *cobra.Command) (*clientcmdapi.Config, error) {
	clientConfig, err := common.GetClientConfig(cmd)
	if err != nil {
		return nil, err
	}
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, err
	}
	return &rawConfig, nil
}
