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

package jobqueuecontroller

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/job"
	executionv1alpha1 "github.com/furiko-io/furiko/pkg/generated/clientset/versioned/typed/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/logvalues"
)

// JobControlInterface is the client interface for the Controller.
type JobControlInterface interface {
	StartJob(ctx context.Context, rj *execution.Job) error
	RejectJob(ctx context.Context, rj *execution.Job, msg string) error
}

// JobControl is the default implementation of JobControlInterface.
type JobControl struct {
	client   executionv1alpha1.ExecutionV1alpha1Interface
	recorder record.EventRecorder
}

var _ JobControlInterface = (*JobControl)(nil)

func NewJobControl(client executionv1alpha1.ExecutionV1alpha1Interface, recorder record.EventRecorder) *JobControl {
	return &JobControl{
		client:   client,
		recorder: recorder,
	}
}

// StartJob sets the startTime of the Job to the current time.
func (c *JobControl) StartJob(ctx context.Context, rj *execution.Job) error {
	newRj := rj.DeepCopy()
	newRj.Status.StartTime = ktime.Now()
	updatedRj, err := c.client.Jobs(rj.GetNamespace()).UpdateStatus(ctx, newRj, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot update job status")
	}

	klog.V(3).InfoS("jobqueuecontroller: started job", logvalues.
		Values("namespace", updatedRj.GetNamespace(), "name", updatedRj.GetName()).
		Level(4, "job", updatedRj).
		Build()...,
	)

	c.recorder.Eventf(rj, corev1.EventTypeNormal, "Started", "Started job successfully")
	return nil
}

func (c *JobControl) RejectJob(ctx context.Context, rj *execution.Job, msg string) error {
	newRj := rj.DeepCopy()
	job.MarkAdmissionError(newRj, msg)

	updatedRj, err := c.client.Jobs(rj.GetNamespace()).Update(ctx, newRj, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot update job")
	}

	klog.V(3).InfoS("jobqueuecontroller: rejected job", logvalues.
		Values("namespace", updatedRj.GetNamespace(), "name", updatedRj.GetName()).
		Level(4, "job", updatedRj).
		Build()...,
	)

	c.recorder.Eventf(rj, corev1.EventTypeWarning, "AdmissionRefused", msg)
	return nil
}
