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
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
)

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
