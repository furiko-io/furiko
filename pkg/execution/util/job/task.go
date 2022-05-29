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

package job

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/execution/util/parallel"
)

// GenerateTaskName generates a deterministic task name given a TaskIndex.
func GenerateTaskName(name string, index tasks.TaskIndex) (string, error) {
	hash, err := parallel.HashIndex(index.Parallel)
	if err != nil {
		return "", errors.Wrapf(err, "cannot hash parallel index")
	}
	return fmt.Sprintf("%v-%v-%v", name, hash, index.Retry), nil
}
