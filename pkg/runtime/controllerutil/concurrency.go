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

package controllerutil

import (
	"runtime"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
)

// GetConcurrencyOrDefaultCPUFactor returns the configured concurrency, or a
// default factor on the number of CPUs.
func GetConcurrencyOrDefaultCPUFactor(concurrency *configv1alpha1.Concurrency, defaultFactor int) int {
	return GetConcurrencyOrDefaultNumber(concurrency, runtime.NumCPU()*defaultFactor)
}

// GetConcurrencyOrDefaultNumber returns the configured concurrency, or a
// default number.
func GetConcurrencyOrDefaultNumber(concurrency *configv1alpha1.Concurrency, defaultNumber int) int {
	if concurrency != nil {
		if num := concurrency.Workers; num > 0 {
			return int(num)
		}
		if factor := concurrency.FactorOfCPUs; factor > 0 {
			return runtime.NumCPU() * int(factor)
		}
	}
	return defaultNumber
}
