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

package leaderelection

import (
	"os"

	"k8s.io/apimachinery/pkg/util/uuid"
)

// GenerateLeaseIdentity generates a unique lease identity that can be used in a leader election.
func GenerateLeaseIdentity() (string, error) {
	id, err := os.Hostname()
	if err != nil {
		return "", err
	}

	// Add a unique identifier so that two processes on the same host don't accidentally both become active
	id += "_" + string(uuid.NewUUID())

	return id, nil
}
