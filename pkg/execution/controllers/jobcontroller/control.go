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

package jobcontroller

import (
	"context"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	executionv1alpha1 "github.com/furiko-io/furiko/pkg/generated/clientset/versioned/typed/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/logvalues"
)

// ExecutionControl is a wrapper around the Execution clientset.
type ExecutionControl struct {
	client executionv1alpha1.ExecutionV1alpha1Interface
	name   string
}

func NewExecutionControl(client executionv1alpha1.ExecutionV1alpha1Interface, name string) *ExecutionControl {
	return &ExecutionControl{
		client: client,
		name:   name,
	}
}

func (c *ExecutionControl) UpdateJob(ctx context.Context, rj, newRj *execution.Job) (bool, error) {
	// No need to update if equal.
	if isEqual, err := IsJobEqual(rj, newRj); err != nil {
		return false, errors.Wrapf(err, "cannot compare job")
	} else if isEqual {
		return false, nil
	}

	updatedRj, err := c.client.Jobs(rj.GetNamespace()).Update(ctx, newRj, metav1.UpdateOptions{})
	if err != nil {
		return false, err
	}

	klog.V(3).InfoS("jobcontroller: updated job", logvalues.
		Values("worker", c.name, "namespace", rj.GetNamespace(), "name", rj.GetName()).
		Level(4, "job", updatedRj).
		Build()...,
	)

	return true, nil
}

func (c *ExecutionControl) UpdateJobStatus(ctx context.Context, rj, newRj *execution.Job) (bool, error) {
	// No need to update if equal.
	if isEqual, err := IsJobStatusEqual(rj, newRj); err != nil {
		return false, errors.Wrapf(err, "cannot compare job")
	} else if isEqual {
		return false, nil
	}

	updatedRj, err := c.client.Jobs(rj.GetNamespace()).UpdateStatus(ctx, newRj, metav1.UpdateOptions{})
	if err != nil {
		return false, err
	}

	klog.V(3).InfoS("jobcontroller: updated job status", logvalues.
		Values("worker", c.name, "namespace", rj.GetNamespace(), "name", rj.GetName()).
		Level(4, "job", updatedRj).
		Build()...,
	)

	return true, nil
}

func (c *ExecutionControl) DeleteJob(ctx context.Context, rj *execution.Job, o metav1.DeleteOptions) error {
	propagation := metav1.DeletePropagationBackground
	if o.PropagationPolicy != nil {
		propagation = *o.PropagationPolicy
	}

	options := *o.DeepCopy()
	options.PropagationPolicy = &propagation

	klog.V(3).InfoS("jobcontroller: deleting job", logvalues.
		Values("worker", c.name, "namespace", rj.GetNamespace(), "name", rj.GetName(),
			"propagation", propagation).
		Level(4, "job", rj).
		Build()...,
	)

	if err := c.client.Jobs(rj.GetNamespace()).Delete(ctx, rj.Name, options); kerrors.IsNotFound(err) {
		// NOTE(irvinlim): Return nil error here if already deleted to suppress errors.
		return nil
	} else if err != nil {
		return err
	}

	klog.V(3).InfoS("jobcontroller: deleted job", logvalues.
		Values("worker", c.name, "namespace", rj.GetNamespace(), "name", rj.GetName(),
			"propagation", propagation).
		Level(4, "job", rj).
		Build()...,
	)

	return nil
}
