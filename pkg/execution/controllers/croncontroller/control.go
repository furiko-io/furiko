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

package croncontroller

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	executionv1alpha1 "github.com/furiko-io/furiko/pkg/generated/clientset/versioned/typed/execution/v1alpha1"
)

type ExecutionControlInterface interface {
	CreateJob(ctx context.Context, rjc *execution.JobConfig, rj *execution.Job) error
}

// ExecutionControl is a wrapper around the Execution clientset.
type ExecutionControl struct {
	name     string
	client   executionv1alpha1.ExecutionV1alpha1Interface
	recorder Recorder
}

var _ ExecutionControlInterface = (*ExecutionControl)(nil)

func NewExecutionControl(
	name string,
	client executionv1alpha1.ExecutionV1alpha1Interface,
	recorder Recorder,
) *ExecutionControl {
	return &ExecutionControl{
		name:     name,
		client:   client,
		recorder: recorder,
	}
}

func (c *ExecutionControl) CreateJob(ctx context.Context, rjc *execution.JobConfig, rj *execution.Job) error {
	createdRj, err := c.client.Jobs(rj.GetNamespace()).Create(ctx, rj, metav1.CreateOptions{})

	// If we fail to create the Job due to an invalid error (e.g. from webhook), do
	// not retry and instead store as an event.
	if kerrors.IsInvalid(err) {
		// Get detailed error information if possible.
		errMessage := err.Error()
		if err, ok := err.(kerrors.APIStatus); ok {
			if details := err.Status().Details; details != nil && len(details.Causes) > 0 {
				causes := make([]string, 0, len(details.Causes))
				for _, cause := range details.Causes {
					causes = append(causes, fmt.Sprintf("%s: %s", cause.Field, cause.Message))
				}
				errMessage = strings.Join(causes, ", ")
			}
		}

		c.recorder.CreateJobFailed(ctx, rjc, rj, errMessage)
		return nil
	}

	if err != nil {
		return errors.Wrapf(err, "cannot create job")
	}

	c.recorder.CreatedJob(ctx, rjc, createdRj)
	return nil
}
