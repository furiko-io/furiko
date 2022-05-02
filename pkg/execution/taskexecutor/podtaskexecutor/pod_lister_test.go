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

package podtaskexecutor_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/taskexecutor/podtaskexecutor"
)

const (
	jobUID          = "922cef24-a2b9-44e5-91e6-bab964401ee7"
	nonJobUID       = "922cef24-a2b9-44e5-91e6-bab964401ee8"
	jobNamespace    = "test"
	nonJobNamespace = "test-2"
)

var (
	fakeJob = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-sample-job",
			Namespace: jobNamespace,
			UID:       jobUID,
		},
		Spec: execution.JobSpec{
			Template: &execution.JobTemplateSpec{
				Task: execution.TaskSpec{
					Template: execution.TaskTemplate{
						Pod: &corev1.PodTemplateSpec{},
					},
				},
			},
		},
	}

	fakePods = []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podtaskexecutor.GetPodIndexedName(fakeJob.Name, 1),
				Namespace: jobNamespace,
				Labels: map[string]string{
					podtaskexecutor.LabelKeyJobUID: jobUID,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podtaskexecutor.GetPodIndexedName(fakeJob.Name, 2),
				Namespace: nonJobNamespace,
				Labels: map[string]string{
					podtaskexecutor.LabelKeyJobUID: jobUID,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podtaskexecutor.GetPodIndexedName(fakeJob.Name, 3),
				Namespace: jobNamespace,
				Labels: map[string]string{
					podtaskexecutor.LabelKeyJobUID: nonJobUID,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podtaskexecutor.GetPodIndexedName(fakeJob.Name, 4),
				Namespace: jobNamespace,
			},
		},
	}
)

func TestNewPodTaskLister(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientset := fake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(clientset, 0)
	lister := podtaskexecutor.NewPodTaskLister(
		informerFactory.Core().V1().Pods().Lister(),
		clientset.CoreV1().Pods(jobNamespace),
		fakeJob,
	)

	// Populate Pods
	createdPods := make([]*corev1.Pod, 0, len(fakePods))
	for _, pod := range fakePods {
		createdPod, err := clientset.CoreV1().Pods(pod.GetNamespace()).Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			t.Errorf("cannot create pod: %v", err)
		}
		createdPods = append(createdPods, createdPod)
	}

	// Start informer
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())

	// Should list only 1 job
	list, err := lister.List()
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, createdPods[0].GetName(), list[0].GetName())

	// Should be able to get task by index
	task, err := lister.Index(1)
	assert.NoError(t, err)
	assert.Equal(t, createdPods[0].GetName(), task.GetName())

	// Returns NotFound error when index not found
	_, err = lister.Index(0)
	assert.True(t, kerrors.IsNotFound(err))
	_, err = lister.Index(2)
	assert.True(t, kerrors.IsNotFound(err))

	// Should be able to get task by name
	task, err = lister.Get(createdPods[0].GetName())
	assert.NoError(t, err)
	assert.Equal(t, createdPods[0].GetName(), task.GetName())

	// Should also be able to get task that don't belong to job
	task, err = lister.Get(createdPods[2].GetName())
	assert.NoError(t, err)
	assert.Equal(t, createdPods[2].GetName(), task.GetName())

	// Cannot get pods outside of namespace
	_, err = lister.Get(createdPods[1].GetName())
	assert.True(t, kerrors.IsNotFound(err))

	// Returns NotFound error when task not found
	_, err = lister.Get("invalid-pod")
	assert.True(t, kerrors.IsNotFound(err))
}
