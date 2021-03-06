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

package testing

import (
	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
)

var (
	// ReconcilerDefaultConcurrency is the default concurrency to use for tests.
	// For simplicity, reconcilers should not have concurrent workers.
	ReconcilerDefaultConcurrency = &configv1alpha1.Concurrency{
		Workers: 1,
	}
)
