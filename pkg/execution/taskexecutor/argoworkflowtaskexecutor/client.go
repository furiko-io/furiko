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

package argoworkflowtaskexecutor

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	coreerrors "github.com/furiko-io/furiko/pkg/core/errors"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
)

type client struct {
	job    *execution.Job
	client v1alpha1.WorkflowInterface
}

var _ tasks.TaskClient = (*client)(nil)

func NewClient(job *execution.Job, wfClient v1alpha1.WorkflowInterface) tasks.TaskClient {
	return &client{
		job:    job,
		client: wfClient,
	}
}

func (c *client) CreateIndex(ctx context.Context, index tasks.TaskIndex) (tasks.Task, error) {
	var template *execution.ArgoWorkflowTemplateSpec
	if jobTemplate := c.job.Spec.Template; jobTemplate != nil && jobTemplate.TaskTemplate.ArgoWorkflow != nil {
		template = jobTemplate.TaskTemplate.ArgoWorkflow
	}
	if template == nil {
		return nil, errors.New("workflow template cannot be empty")
	}
	newWf, err := NewWorkflow(c.job, template, index)
	if err != nil {
		return nil, err
	}

	// Create new workflow.
	wf, err := c.client.Create(ctx, newWf, metav1.CreateOptions{})

	// Rejected by apiserver, do not attempt to retry and raise an
	// AdmissionRefusedError instead.
	if kerrors.IsInvalid(err) {
		return nil, coreerrors.NewAdmissionRefusedError(err.Error())
	}

	if err != nil {
		return nil, errors.Wrapf(err, "cannot create workflow")
	}

	return NewTask(wf), nil
}

func (c *client) Delete(ctx context.Context, name string, force bool) error {
	opts := metav1.DeleteOptions{}
	if force {
		opts.GracePeriodSeconds = pointer.Int64(0)
	}
	if err := c.client.Delete(ctx, name, opts); err != nil {
		return errors.Wrapf(err, "could not delete workflow")
	}
	return nil
}
