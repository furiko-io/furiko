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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobcontroller"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/cmp"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

func TestExecutionControl(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	ktime.Clock = clock.NewFakeClock(time.Now())
	c := mock.NewContext()
	client := c.MockClientsets().FurikoMock().ExecutionV1alpha1()
	control := jobcontroller.NewExecutionControl(client, "test")
	err := c.Start(ctx)
	assert.NoError(t, err)

	// Cannot update non-existent job
	newJob := fakeJob.DeepCopy()
	newJob.Spec.KillTimestamp = ktime.Now()
	newJob.Status.Phase = execution.JobRunning
	_, err = control.UpdateJob(ctx, fakeJob, newJob)
	assert.True(t, kerrors.IsNotFound(err))
	_, err = control.UpdateJobStatus(ctx, fakeJob, newJob)
	assert.True(t, kerrors.IsNotFound(err))

	// Deleting non-existent job returns no error
	err = control.DeleteJob(ctx, fakeJob.DeepCopy(), metav1.DeleteOptions{})
	assert.NoError(t, err)

	// Populate job
	createdJob, err := client.Jobs(fakeJob.Namespace).Create(ctx, fakeJob, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Not updated if equal
	newJob = createdJob.DeepCopy()
	updated, err := control.UpdateJob(ctx, fakeJob, newJob)
	assert.NoError(t, err)
	assert.False(t, updated)

	// Update should succeed
	newJob.Spec.KillTimestamp = ktime.Now()
	updated, err = control.UpdateJob(ctx, fakeJob, newJob)
	assert.NoError(t, err)
	assert.True(t, updated)

	// Ensure it was updated
	job, err := client.Jobs(fakeJob.Namespace).Get(ctx, fakeJob.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	equal, err := cmp.IsJSONEqual(job, newJob)
	assert.NoError(t, err)
	assert.True(t, equal)

	// Not updated if equal
	newJob = fakeJob.DeepCopy()
	updated, err = control.UpdateJobStatus(ctx, fakeJob, newJob)
	assert.NoError(t, err)
	assert.False(t, updated)

	// Update status should succeed
	newJob = fakeJob.DeepCopy()
	newJob.Status.Phase = execution.JobRunning
	updated, err = control.UpdateJobStatus(ctx, fakeJob, newJob)
	assert.NoError(t, err)
	assert.True(t, updated)

	// Ensure it was updated
	job, err = client.Jobs(fakeJob.Namespace).Get(ctx, fakeJob.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	equal, err = cmp.IsJSONEqual(job, newJob)
	assert.NoError(t, err)
	assert.True(t, equal)

	// Delete should succeed
	err = control.DeleteJob(ctx, fakeJob, metav1.DeleteOptions{})
	assert.NoError(t, err)
}
