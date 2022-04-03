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

package podtaskexecutor

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	rerrors "github.com/furiko-io/furiko/pkg/errors"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
)

// PodTaskClient operates on Pod tasks.
type PodTaskClient struct {
	client v1.PodInterface
	rj     *execution.Job
}

func NewPodTaskClient(client v1.PodInterface, rj *execution.Job) *PodTaskClient {
	return &PodTaskClient{
		client: client,
		rj:     rj,
	}
}

func (p *PodTaskClient) Get(ctx context.Context, name string) (tasks.Task, error) {
	pod, err := p.client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pod")
	}
	return p.new(pod), nil
}

func (p *PodTaskClient) Index(ctx context.Context, index int64) (tasks.Task, error) {
	return p.Get(ctx, GetPodIndexedName(p.rj.GetName(), index))
}

func (p *PodTaskClient) CreateIndex(ctx context.Context, index int64) (tasks.Task, error) {
	// Create pod object
	newPod, err := NewPod(p.rj, index)
	if err != nil {
		return nil, err
	}

	// Create resource
	pod, err := p.client.Create(ctx, newPod, metav1.CreateOptions{})

	// Rejected by apiserver, do not attempt to retry and raise an
	// AdmissionRefusedError instead.
	if kerrors.IsInvalid(err) {
		return nil, rerrors.NewAdmissionRefusedError(err.Error())
	}

	if err != nil {
		return nil, errors.Wrapf(err, "could not create pod")
	}

	return p.new(pod), nil
}

func (p *PodTaskClient) Delete(ctx context.Context, name string, force bool) error {
	opts := metav1.DeleteOptions{}

	// Force delete pod using grace period set as 0.
	if force {
		var grace int64
		opts.GracePeriodSeconds = &grace
	}

	if err := p.client.Delete(ctx, name, opts); err != nil {
		return errors.Wrapf(err, "could not delete pod")
	}

	return nil
}

func (p *PodTaskClient) new(pod *corev1.Pod) tasks.Task {
	return NewPodTask(pod, p.client)
}
