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

package jobcontroller_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobcontroller"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

func TestIsJobEqual(t *testing.T) {
	// Test DeepCopy.
	orig := &execution.Job{
		Status: execution.JobStatus{
			Phase: execution.JobStarting,
		},
	}
	updated := orig.DeepCopy()
	isEqual, err := jobcontroller.IsJobEqual(orig, updated)
	assert.NoError(t, err)
	assert.True(t, isEqual, "should be equal, diff = %v", cmp.Diff(orig, updated))

	// Test updating field manually.
	updated.Spec.KillTimestamp = ktime.Now()
	isEqual, err = jobcontroller.IsJobEqual(orig, updated)
	assert.NoError(t, err)
	assert.False(t, isEqual)
}
