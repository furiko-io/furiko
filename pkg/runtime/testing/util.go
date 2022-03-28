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
	"fmt"

	ktesting "k8s.io/client-go/testing"
)

// GetFullResourceName returns the full resource name, including subresource if any.
func GetFullResourceName(action ktesting.Action) string {
	if action.GetSubresource() != "" {
		return fmt.Sprintf("%v/%v", action.GetResource().Resource, action.GetSubresource())
	}
	return action.GetResource().Resource
}
