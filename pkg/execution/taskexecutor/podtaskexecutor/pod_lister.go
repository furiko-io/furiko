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
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	jobtasks "github.com/furiko-io/furiko/pkg/execution/tasks"
)

// PodTaskLister lists Pod tasks.
type PodTaskLister struct {
	podLister corev1lister.PodLister
	client    v1.PodInterface
	rj        *execution.Job
}

func NewPodTaskLister(
	podLister corev1lister.PodLister, client v1.PodInterface, rj *execution.Job,
) *PodTaskLister {
	return &PodTaskLister{
		podLister: podLister,
		client:    client,
		rj:        rj,
	}
}

func (p *PodTaskLister) Get(name string) (jobtasks.Task, error) {
	pod, err := p.podLister.Pods(p.rj.GetNamespace()).Get(name)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get task")
	}
	return p.new(pod), nil
}

func (p *PodTaskLister) Index(index int64) (jobtasks.Task, error) {
	return p.Get(GetPodIndexedName(p.rj.GetName(), index))
}

func (p *PodTaskLister) List() ([]jobtasks.Task, error) {
	selector := labels.SelectorFromSet(LabelPodsForJob(p.rj))
	pods, err := p.podLister.Pods(p.rj.GetNamespace()).List(selector)
	if err != nil {
		return nil, errors.Wrapf(err, "could not list pods")
	}
	tasks := make([]jobtasks.Task, 0, len(pods))
	for _, pod := range pods {
		tasks = append(tasks, p.new(pod))
	}
	return tasks, nil
}

func (p *PodTaskLister) new(pod *corev1.Pod) jobtasks.Task {
	return NewPodTask(pod, p.client)
}
