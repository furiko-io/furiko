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

package completion_test

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

var (
	defaultJobConfigReadyEnabled = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jobconfig-ready-enabled",
			Namespace: DefaultNamespace,
		},
		Status: execution.JobConfigStatus{
			State: execution.JobConfigReadyEnabled,
		},
	}

	defaultJobConfigLastExecuted = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jobconfig-last-executed",
			Namespace: DefaultNamespace,
		},
		Status: execution.JobConfigStatus{
			State:        execution.JobConfigReadyEnabled,
			LastExecuted: testutils.Mkmtimep(executeTime),
			Active:       1,
		},
	}

	prodJobConfig = &execution.JobConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod-jobconfig",
			Namespace: ProdNamespace,
		},
		Status: execution.JobConfigStatus{
			State: execution.JobConfigReadyEnabled,
		},
	}
)

func TestListJobConfigsCompleter(t *testing.T) {
	runtimetesting.RunCompleterTests(t, []runtimetesting.CompleterTest{
		{
			Name:      "no jobconfigs",
			Completer: &completion.ListJobConfigsCompleter{},
			Want:      []string{},
		},
		{
			Name: "show single job config",
			Fixtures: []runtime.Object{
				defaultJobConfigReadyEnabled,
			},
			Completer: &completion.ListJobConfigsCompleter{},
			Want: []string{
				`jobconfig-ready-enabled	ReadyEnabled`,
			},
		},
		{
			Name: "show single job config with last executed",
			Fixtures: []runtime.Object{
				defaultJobConfigLastExecuted,
			},
			Now:       testutils.Mktime(currentTime),
			Completer: &completion.ListJobConfigsCompleter{},
			Want: []string{
				`jobconfig-last-executed	ReadyEnabled, last executed Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
			},
		},
		{
			Name: "show default namespace",
			Fixtures: []runtime.Object{
				defaultJobConfigReadyEnabled,
				prodJobConfig,
			},
			Namespace: DefaultNamespace,
			Now:       testutils.Mktime(currentTime),
			Completer: &completion.ListJobConfigsCompleter{},
			Want: []string{
				`jobconfig-ready-enabled	ReadyEnabled`,
			},
		},
		{
			Name: "show prod namespace",
			Fixtures: []runtime.Object{
				defaultJobConfigReadyEnabled,
				prodJobConfig,
			},
			Namespace: ProdNamespace,
			Now:       testutils.Mktime(currentTime),
			Completer: &completion.ListJobConfigsCompleter{},
			Want: []string{
				`prod-jobconfig	ReadyEnabled`,
			},
		},
		{
			Name: "sort by alphabetical order",
			Fixtures: []runtime.Object{
				defaultJobConfigReadyEnabled,
				defaultJobConfigLastExecuted,
			},
			Now:       testutils.Mktime(currentTime),
			Completer: &completion.ListJobConfigsCompleter{},
			Want: []string{
				`jobconfig-last-executed	ReadyEnabled, last executed Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
				`jobconfig-ready-enabled	ReadyEnabled`,
			},
		},
		{
			Name: "with sort",
			Fixtures: []runtime.Object{
				defaultJobConfigReadyEnabled,
				defaultJobConfigLastExecuted,
			},
			Now: testutils.Mktime(currentTime),
			Completer: &completion.ListJobConfigsCompleter{
				Sort: func(a, b *execution.JobConfig) bool {
					return a.Name > b.Name
				},
			},
			Want: []string{
				`jobconfig-ready-enabled	ReadyEnabled`,
				`jobconfig-last-executed	ReadyEnabled, last executed Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
			},
		},
		{
			Name: "with filter",
			Fixtures: []runtime.Object{
				defaultJobConfigReadyEnabled,
				defaultJobConfigLastExecuted,
			},
			Now: testutils.Mktime(currentTime),
			Completer: &completion.ListJobConfigsCompleter{
				Filter: func(job *execution.JobConfig) bool {
					return !job.Status.LastExecuted.IsZero()
				},
			},
			Want: []string{
				`jobconfig-last-executed	ReadyEnabled, last executed Tue, 09 Feb 2021 04:02:00 +00:00 (6 hours ago)`,
			},
		},
		{
			Name: "with format",
			Fixtures: []runtime.Object{
				defaultJobConfigReadyEnabled,
				defaultJobConfigLastExecuted,
			},
			Completer: &completion.ListJobConfigsCompleter{
				Format: func(job *execution.JobConfig) string {
					return fmt.Sprintf("%v\t%v active jobs", job.Name, job.Status.Active)
				},
			},
			Want: []string{
				`jobconfig-last-executed	1 active jobs`,
				`jobconfig-ready-enabled	0 active jobs`,
			},
		},
	})
}
