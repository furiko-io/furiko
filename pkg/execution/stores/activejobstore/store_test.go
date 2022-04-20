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

package activejobstore_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/stores/activejobstore"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
	furiko "github.com/furiko-io/furiko/pkg/generated/clientset/versioned"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/utils/meta"
)

const (
	fakeclientsetSleepDuration = time.Millisecond * 10
	testNamespace              = "test"
	mockStartTime              = "2021-02-09T04:06:09Z"
)

var (
	stdStartTime, _ = time.Parse(time.RFC3339, mockStartTime)
	startTime       = metav1.NewTime(stdStartTime)

	rjc1 = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "jobconfig1",
			UID:       "3addee29-aecb-499e-a3ef-9b57919be167",
		},
	}

	rjc2 = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "jobconfig2",
			UID:       "6a03e87c-2b1a-4756-b594-0dfd8539705f",
		},
	}

	rjc1Namespace = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "mycustomnamespace",
			Name:      "jobconfig1",
			UID:       "87265b24-78d8-4913-9dfa-813d2efd5a51",
		},
	}
)

func TestStore(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrlContext := mock.NewContext()
	client := ctrlContext.MockClientsets().FurikoMock()
	err := ctrlContext.Start(ctx)
	assert.NoError(t, err)
	store, err := activejobstore.NewStore(ctrlContext)
	assert.NoError(t, err)

	// Add some Jobs to store
	rjs := []*execution.Job{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "jobconfig1-job1",
				Labels: map[string]string{
					jobconfig.LabelKeyJobConfigUID: string(rjc1.UID),
				},
			},
			Status: execution.JobStatus{
				StartTime: &startTime,
				Phase:     execution.JobStarting,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "jobconfig1-job2",
				Labels: map[string]string{
					jobconfig.LabelKeyJobConfigUID: string(rjc1.UID),
				},
			},
			Status: execution.JobStatus{
				StartTime: &startTime,
				Phase:     execution.JobSucceeded,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "jobconfig2-job1",
				Labels: map[string]string{
					jobconfig.LabelKeyJobConfigUID: string(rjc2.UID),
				},
			},
			Status: execution.JobStatus{
				StartTime: &startTime,
				Phase:     execution.JobSucceeded,
			},
		},
	}
	for _, rj := range rjs {
		_, err := createJob(ctx, client, rj)
		assert.NoError(t, err)
	}

	// Add some JobConfigs to store
	rjcs := []*execution.JobConfig{rjc1, rjc2}
	for _, rjc := range rjcs {
		_, err := client.ExecutionV1alpha1().JobConfigs(rjc.Namespace).Create(ctx, rjc, metav1.CreateOptions{})
		assert.NoError(t, err)
	}

	// Recover store
	err = store.Recover(ctx)
	assert.NoError(t, err)

	// Check store contents
	assert.Equal(t, int64(1), store.CountActiveJobsForConfig(rjc1))
	assert.Equal(t, int64(0), store.CountActiveJobsForConfig(rjc2))

	// Before creating new Job, need to use CheckAndAdd (we set StartTime explicitly
	// here as a shortcut)
	rj := &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "jobconfig2-job2",
			Labels: map[string]string{
				jobconfig.LabelKeyJobConfigUID: string(rjc2.UID),
			},
		},
		Status: execution.JobStatus{
			StartTime: &startTime,
		},
	}
	for !store.CheckAndAdd(rjc2, store.CountActiveJobsForConfig(rjc2)) {
	}
	createdRj, err := createJob(ctx, client, rj)
	assert.NoError(t, err)
	rj = createdRj

	// Check store contents
	assert.Equal(t, int64(1), store.CountActiveJobsForConfig(rjc1))
	assert.Equal(t, int64(1), store.CountActiveJobsForConfig(rjc2))

	// Update to inactive a few times.
	// This is to test for negative counter invariant.
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(i int) {
			defer wg.Done()
			newRj := rj.DeepCopy()
			meta.SetAnnotation(newRj, "index", strconv.Itoa(i))
			newRj.Status.Phase = execution.JobSucceeded
			updatedRj, err := updateJob(ctx, client, newRj)
			assert.NoError(t, err)
			rj = updatedRj

			// Check store contents
			assert.Equal(t, int64(1), store.CountActiveJobsForConfig(rjc1))
			assert.Equal(t, int64(0), store.CountActiveJobsForConfig(rjc2))
		}(i)
	}
	wg.Wait()

	// Update back to active.
	newRj := rj.DeepCopy()
	newRj.Status.Phase = execution.JobStarting
	updatedRj, err := updateJob(ctx, client, newRj)
	assert.NoError(t, err)
	newRj = updatedRj

	// Check store contents
	assert.Equal(t, int64(1), store.CountActiveJobsForConfig(rjc1))
	assert.Equal(t, int64(1), store.CountActiveJobsForConfig(rjc2))

	// Delete Job.
	err = deleteJob(ctx, client, newRj)
	assert.NoError(t, err)

	// Check store contents
	assert.Equal(t, int64(1), store.CountActiveJobsForConfig(rjc1))
	assert.Equal(t, int64(0), store.CountActiveJobsForConfig(rjc2))

	// Test Job with same name in a different namespace (will have different UID)
	assert.Equal(t, int64(1), store.CountActiveJobsForConfig(rjc1))
	assert.Equal(t, int64(0), store.CountActiveJobsForConfig(rjc2))
	assert.Equal(t, int64(0), store.CountActiveJobsForConfig(rjc1Namespace))
}

func TestStore_CheckAndAdd_NotStarted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrlContext := mock.NewContext()
	client := ctrlContext.MockClientsets().FurikoMock()
	err := ctrlContext.Start(ctx)
	assert.NoError(t, err)
	store, err := activejobstore.NewStore(ctrlContext)
	assert.NoError(t, err)
	err = store.Recover(ctx)
	assert.NoError(t, err)

	// Create job without starting it.
	rj := &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "jobconfig2-job2",
			Labels: map[string]string{
				jobconfig.LabelKeyJobConfigUID: string(rjc1.UID),
			},
		},
	}
	createdRj, err := createJob(ctx, client, rj)
	assert.NoError(t, err)
	rj = createdRj

	// Perform CheckAndAdd of previously unstarted job.
	// This procedure is done in the JobQueueController.
	for {
		count := store.CountActiveJobsForConfig(rjc1)
		if !store.CheckAndAdd(rjc1, count) {
			continue
		}
		newRj := rj.DeepCopy()
		now := metav1.Now()
		newRj.Status.StartTime = &now
		if _, err := updateJob(ctx, client, newRj); err == nil {
			break
		}
	}

	// Ensure that count is still 1.
	assert.Equal(t, int64(1), store.CountActiveJobsForConfig(rjc1))

	// Delete the key, ensure that it drops back to 0.
	store.Delete(rjc1)
	assert.Equal(t, int64(0), store.CountActiveJobsForConfig(rjc1))
}

func createJob(ctx context.Context, client furiko.Interface, rj *execution.Job) (*execution.Job, error) {
	created, err := client.ExecutionV1alpha1().Jobs(rj.Namespace).Create(ctx, rj, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	time.Sleep(fakeclientsetSleepDuration)
	return created, nil
}

func updateJob(ctx context.Context, client furiko.Interface, rj *execution.Job) (*execution.Job, error) {
	updatedRj, err := client.ExecutionV1alpha1().Jobs(rj.Namespace).Update(ctx, rj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	time.Sleep(fakeclientsetSleepDuration)
	return updatedRj, nil
}

func deleteJob(ctx context.Context, client furiko.Interface, rj *execution.Job) error {
	err := client.ExecutionV1alpha1().Jobs(rj.Namespace).Delete(ctx, rj.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	time.Sleep(fakeclientsetSleepDuration)
	return nil
}
