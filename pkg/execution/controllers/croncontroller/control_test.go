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

package croncontroller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ktesting "k8s.io/client-go/testing"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/croncontroller"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

const (
	jobNamespace = "test"
	jobName      = "my-sample-job"
)

var (
	fakeJob = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: jobNamespace,
		},
		Spec: execution.JobSpec{
			Type: execution.JobTypeAdhoc,
			Template: &execution.JobTemplate{
				TaskTemplate: execution.TaskTemplate{
					Pod: &execution.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "container",
									Image: "hello-world",
									Args: []string{
										"echo",
										"Hello world!",
									},
								},
							},
						},
					},
				},
			},
		},
	}
)

func TestExecutionControl(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	ktime.Clock = clock.NewFakeClock(time.Now())
	c := mock.NewContext()
	fakeClient := c.MockClientsets().FurikoMock()
	client := fakeClient.ExecutionV1alpha1()
	recorder := newFakeRecorder()
	control := croncontroller.NewExecutionControl("test", client, recorder)
	err := c.Start(ctx)
	assert.NoError(t, err)

	// Create job with client with recorded event
	assert.Len(t, recorder.Events, 0)
	err = control.CreateJob(ctx, jobConfigAllow, fakeJob)
	assert.NoError(t, err)
	assert.Len(t, recorder.Events, 1)
	assert.Equal(t, "CreatedJob", recorder.Events[0])
	recorder.Clear()

	// Should be created
	_, err = client.Jobs(fakeJob.Namespace).Get(ctx, fakeJob.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// Creating duplicate Job should throw errpr
	err = control.CreateJob(ctx, jobConfigAllow, fakeJob)
	assert.True(t, kerrors.IsAlreadyExists(err))

	// Delete the Job now
	err = client.Jobs(fakeJob.Namespace).Delete(ctx, fakeJob.Name, metav1.DeleteOptions{})
	assert.NoError(t, err)

	// Add Reactor to simulate InvalidError
	fakeClient.PrependReactor("create", "jobs",
		func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, kerrors.NewInvalid(execution.GVKJob.GroupKind(), fakeJob.Name, field.ErrorList{
				field.Invalid(field.NewPath("metadata", "name"), fakeJob.Name, "invalid name"),
			})
		})

	// Create Job with InvalidError, should not throw error but recorded an event
	assert.Len(t, recorder.Events, 0)
	err = control.CreateJob(ctx, jobConfigAllow, fakeJob)
	assert.NoError(t, err)
	assert.Len(t, recorder.Events, 1)
	assert.Equal(t, "CreateJobFailed", recorder.Events[0])

	// Should not be created
	_, err = client.Jobs(fakeJob.Namespace).Get(ctx, fakeJob.Name, metav1.GetOptions{})
	assert.Error(t, err)
	assert.True(t, kerrors.IsNotFound(err))
}
