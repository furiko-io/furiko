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

package v1alpha1

// ConfigName refers to the name of a dynamic configuration.
type ConfigName string

const (
	// JobExecutionConfigName refers to JobExecutionConfig.
	JobExecutionConfigName ConfigName = "execution-job"

	// CronExecutionConfigName refers to CronExecutionConfig.
	CronExecutionConfigName ConfigName = "execution-cron"
)
