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
	executiongroup "github.com/furiko-io/furiko/apis/execution"
)

var (
	// LabelKeyAdmissionErrorMessage stores any user error encountered when creating
	// tasks. This usually implies a misconfiguration that we cannot retry further,
	// hence the Job should transit into a terminal state.
	LabelKeyAdmissionErrorMessage = executiongroup.AddGroupToLabel("admission-error")
)
