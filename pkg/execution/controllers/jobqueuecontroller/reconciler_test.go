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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/execution/jobconfig"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

const (
	jobNamespace = "test"
	createTime   = "2021-02-09T04:06:00Z"
	now          = "2021-02-09T04:06:05Z"
	startAfter   = "2021-02-09T05:00:00Z"
	uid1         = "0ed1bc76-07ca-4cf7-9a47-a0cc4aec48b9"
	uid2         = "6e08ee33-ccbe-4fc5-9c46-e29c19cc2fcb"
)

var (
	resourceJob = runtimetesting.NewGroupVersionResource(executiongroup.GroupName, execution.Version, "jobs")
	timeNow     = testutils.Mkmtimep(now)

	startPolicy = &execution.StartPolicySpec{
		StartAfter: testutils.Mkmtimep(startAfter),
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
			UID:       uid1,
			Namespace: jobNamespace,
			Name:      "job-config-1",
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
				jobconfig.LabelKeyJobConfigUID: uid1,
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
				jobconfig.LabelKeyJobConfigUID: uid2,
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
				jobconfig.LabelKeyJobConfigUID: uid1,
			},
		},
		Spec: execution.JobSpec{
			StartPolicy: startPolicy,
		},
	}
)

func startJob(job *execution.Job, now *metav1.Time) *execution.Job {
	newJob := job.DeepCopy()
	newJob.Status.StartTime = now
	return newJob
}
