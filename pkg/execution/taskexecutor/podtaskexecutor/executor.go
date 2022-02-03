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
	coreinformers "k8s.io/client-go/informers/core/v1"
	corev1clientset "k8s.io/client-go/kubernetes/typed/core/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/tasks"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

type executor struct {
	rj       *execution.Job
	informer coreinformers.PodInformer
	client   corev1clientset.CoreV1Interface
}

// NewExecutor returns a new tasks.Executor which lists and operates on Pods.
func NewExecutor(
	clientsets controllercontext.Clientsets, informers controllercontext.Informers, rj *execution.Job,
) tasks.Executor {
	return &executor{
		rj:       rj,
		informer: informers.Kubernetes().Core().V1().Pods(),
		client:   clientsets.Kubernetes().CoreV1(),
	}
}

func (e *executor) Lister() tasks.TaskLister {
	podLister := e.informer.Lister()
	return NewPodTaskLister(podLister, e.client.Pods(e.rj.GetNamespace()), e.rj)
}

func (e *executor) Client() tasks.TaskClient {
	return NewPodTaskClient(e.client.Pods(e.rj.GetNamespace()), e.rj)
}

type factory struct {
	clientsets controllercontext.Clientsets
	informers  controllercontext.Informers
}

// NewFactory returns a new tasks.ExecutorFactory to return a Pod task executor.
func NewFactory(clientsets controllercontext.Clientsets, informers controllercontext.Informers) tasks.ExecutorFactory {
	return &factory{
		clientsets: clientsets,
		informers:  informers,
	}
}

func (f *factory) ForJob(rj *execution.Job) (tasks.Executor, error) {
	return NewExecutor(f.clientsets, f.informers, rj), nil
}
