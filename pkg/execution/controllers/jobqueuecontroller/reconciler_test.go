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

package jobqueuecontroller_test

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/controllers/jobqueuecontroller"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/execution/util/jobconfig"
	executionv1alpha1 "github.com/furiko-io/furiko/pkg/generated/clientset/versioned/typed/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	jobNamespace = "test"
	createTime   = "2021-02-09T04:06:00Z"
	now          = "2021-02-09T04:06:05Z"
	startAfter   = "2021-02-09T05:00:00Z"

	uidJobConfig   = "0ed1bc76-07ca-4cf7-9a47-a0cc4aec48b9"
	uidForbid      = "25b6cc22-a0c0-4f55-97b8-b798769d462b"
	uidNonexistent = "6e08ee33-ccbe-4fc5-9c46-e29c19cc2fcb"
)

var (
	timeNow = testutils.Mkmtimep(now)

	startPolicy = &execution.StartPolicySpec{
		ConcurrencyPolicy: execution.ConcurrencyPolicyAllow,
		StartAfter:        testutils.Mkmtimep(startAfter),
	}

	jobToBeStarted = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "job-to-be-started",
			Namespace:         jobNamespace,
			CreationTimestamp: testutils.Mkmtime(createTime),
			Finalizers: []string{
				executiongroup.DeleteDependentsFinalizer,
			},
		},
	}

	jobWithStartAfter = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "job-with-start-after",
			Namespace:         jobNamespace,
			CreationTimestamp: testutils.Mkmtime(createTime),
			Finalizers: []string{
				executiongroup.DeleteDependentsFinalizer,
			},
		},
		Spec: execution.JobSpec{
			StartPolicy: startPolicy,
		},
	}

	jobConfig1 = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			UID:       uidJobConfig,
			Namespace: jobNamespace,
			Name:      "job-config-1",
		},
	}

	jobConfigForbid = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			UID:       uidForbid,
			Namespace: jobNamespace,
			Name:      "job-config-forbid",
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy: execution.ConcurrencyPolicyForbid,
			},
		},
	}

	jobConfigForbidMax2 = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			UID:       uidForbid,
			Namespace: jobNamespace,
			Name:      "job-config-forbid",
		},
		Spec: execution.JobConfigSpec{
			Concurrency: execution.ConcurrencySpec{
				Policy:         execution.ConcurrencyPolicyForbid,
				MaxConcurrency: pointer.Int64(2),
			},
		},
	}

	jobForConfig1ToBeStarted = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "job-for-config-1-to-be-started",
			Namespace:         jobNamespace,
			CreationTimestamp: testutils.Mkmtime(createTime),
			Finalizers: []string{
				executiongroup.DeleteDependentsFinalizer,
			},
			Labels: map[string]string{
				jobconfig.LabelKeyJobConfigUID: uidJobConfig,
			},
		},
	}

	jobForConfig2ToBeStarted = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "job-for-config-2-to-be-started",
			Namespace:         jobNamespace,
			CreationTimestamp: testutils.Mkmtime(createTime),
			Finalizers: []string{
				executiongroup.DeleteDependentsFinalizer,
			},
			Labels: map[string]string{
				jobconfig.LabelKeyJobConfigUID: uidNonexistent,
			},
		},
	}

	jobForConfig1WithStartAfter = &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "job-for-config-1-with-start-after",
			Namespace:         jobNamespace,
			CreationTimestamp: testutils.Mkmtime(createTime),
			Finalizers: []string{
				executiongroup.DeleteDependentsFinalizer,
			},
			Labels: map[string]string{
				jobconfig.LabelKeyJobConfigUID: uidJobConfig,
			},
		},
		Spec: execution.JobSpec{
			StartPolicy: startPolicy,
		},
	}

	jobForForbid1 = makeJob("job-for-forbid-1", 0, jobConfigForbid, execution.ConcurrencyPolicyForbid, nil)
	jobForForbid2 = makeJob("job-for-forbid-2", 1, jobConfigForbid, execution.ConcurrencyPolicyForbid, nil)
)

func makeJob(
	name string,
	index int,
	jobConfig *execution.JobConfig,
	cp execution.ConcurrencyPolicy,
	startAfter *metav1.Time,
) *execution.Job {
	return &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: jobNamespace,
			// NOTE(irvinlim): We use a staggered CreationTimestamp because the reconciler
			// orders Jobs using this field internally, so we can trigger sync with
			// deterministic order.
			CreationTimestamp: metav1.NewTime(testutils.Mkmtime(createTime).Add(time.Second * time.Duration(index))),
			Finalizers: []string{
				executiongroup.DeleteDependentsFinalizer,
			},
			Labels: map[string]string{
				jobconfig.LabelKeyJobConfigUID: string(jobConfig.UID),
			},
		},
		Spec: execution.JobSpec{
			StartPolicy: &execution.StartPolicySpec{
				ConcurrencyPolicy: cp,
				StartAfter:        startAfter,
			},
		},
	}
}

func startJob(job *execution.Job, now *metav1.Time) *execution.Job {
	newJob := job.DeepCopy()
	newJob.Status.StartTime = now
	return newJob
}

func rejectJob(job *execution.Job) *execution.Job {
	newJob := job.DeepCopy()
	jobutil.MarkAdmissionError(newJob, "rejected")
	return newJob
}

type mockJobControl struct {
	client executionv1alpha1.ExecutionV1alpha1Interface
}

func newMockJobControl(client executionv1alpha1.ExecutionV1alpha1Interface) *mockJobControl {
	return &mockJobControl{
		client: client,
	}
}

var _ jobqueuecontroller.JobControlInterface = (*mockJobControl)(nil)

func (m *mockJobControl) StartJob(ctx context.Context, rj *execution.Job) error {
	newJob := startJob(rj, ktime.Now())
	_, err := m.client.Jobs(rj.Namespace).Update(ctx, newJob, metav1.UpdateOptions{})
	return err
}

func (m *mockJobControl) RejectJob(ctx context.Context, rj *execution.Job, _ string) error {
	newJob := rejectJob(rj)
	_, err := m.client.Jobs(rj.Namespace).Update(ctx, newJob, metav1.UpdateOptions{})
	return err
}
