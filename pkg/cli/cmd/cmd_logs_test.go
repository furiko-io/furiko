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

package cmd_test

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/cmd"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

func TestLogsCommand(t *testing.T) {
	runtimetesting.RunCommandTests(t, []runtimetesting.CommandTest{
		{
			Name: "display help",
			Args: []string{"logs", "--help"},
			Stdout: runtimetesting.Output{
				Contains: cmd.LogsExample,
			},
		},
		{
			Name:      "need an argument",
			Args:      []string{"logs"},
			WantError: assert.Error,
		},
		{
			Name: "completion",
			Args: []string{cobra.ShellCompRequestCmd, "logs", ""},
			Fixtures: []runtime.Object{
				jobRunning,
				jobFinished,
			},
			Stdout: runtimetesting.Output{
				ContainsAll: []string{
					jobRunning.Name,
					jobFinished.Name,
				},
			},
		},
		{
			Name:      "job not found",
			Args:      []string{"logs", "job-running"},
			WantError: testutils.AssertErrorIsNotFound(),
		},
		{
			Name:     "get logs for job",
			Args:     []string{"logs", "job-running"},
			Fixtures: []runtime.Object{jobRunning},
			Stdout: runtimetesting.Output{
				Exact: "fake logs",
			},
		},
		{
			Name:      "cannot get logs for job without running tasks",
			Args:      []string{"logs", "job-queued"},
			Fixtures:  []runtime.Object{jobQueued},
			WantError: testutils.AssertErrorContains("no tasks created yet"),
		},
		{
			Name:      "cannot get logs without using --index for a parallel job",
			Args:      []string{"logs", "job-parallel"},
			Fixtures:  []runtime.Object{jobParallel},
			WantError: testutils.AssertErrorContains("--index"),
		},
		{
			Name:      "invalid --index",
			Args:      []string{"logs", "job-parallel", "--index", "{}"},
			Fixtures:  []runtime.Object{jobParallel},
			WantError: assert.Error,
		},
		{
			Name:     "use --index to get logs for a parallel job",
			Args:     []string{"logs", "job-parallel", "--index=0"},
			Fixtures: []runtime.Object{jobParallel},
			Stdout: runtimetesting.Output{
				Exact: "fake logs",
			},
		},
		{
			Name:      "cannot get logs when using out of range --index for a non-parallel job",
			Args:      []string{"logs", "job-running", "--index=1"},
			Fixtures:  []runtime.Object{jobRunning},
			WantError: testutils.AssertErrorContains("no started tasks found"),
		},
		{
			Name:      "cannot get logs when using out of range --index for a parallel job",
			Args:      []string{"logs", "job-parallel", "--index=42"},
			Fixtures:  []runtime.Object{jobParallel},
			WantError: testutils.AssertErrorContains("no started tasks found"),
		},
		{
			Name:      "invalid --retry-index",
			Args:      []string{"logs", "job-parallel", "--index", "asd"},
			Fixtures:  []runtime.Object{jobParallel},
			WantError: assert.Error,
		},
		{
			Name:      "cannot get logs when --retry-index is out of range for non-parallel job",
			Args:      []string{"logs", "job-running", "--retry-index=2"},
			Fixtures:  []runtime.Object{jobRunning},
			WantError: testutils.AssertErrorContains("no started tasks found"),
		},
		{
			Name:      "cannot get logs when --retry-index is out of range for parallel job",
			Args:      []string{"logs", "job-parallel", "--index=0", "--retry-index=2"},
			Fixtures:  []runtime.Object{jobParallel},
			WantError: testutils.AssertErrorContains("no started tasks found"),
		},
		{
			Name:      "cannot get logs when --retry-index is negative",
			Args:      []string{"logs", "job-parallel", "--index=0", "--retry-index=-1"},
			Fixtures:  []runtime.Object{jobParallel},
			WantError: testutils.AssertErrorContains("invalid"),
		},
	})
}

func TestParseParallelIndex(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    *execution.ParallelIndex
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "empty string",
			value:   "",
			wantErr: assert.Error,
		},
		{
			name:  "parse as indexNumber",
			value: "0",
			want: &execution.ParallelIndex{
				IndexNumber: pointer.Int64(0),
			},
		},
		{
			name:  "parse as indexKey",
			value: "abc",
			want: &execution.ParallelIndex{
				IndexKey: "abc",
			},
		},
		{
			name:  "parse as matrixValues",
			value: "key=value",
			want: &execution.ParallelIndex{
				MatrixValues: map[string]string{
					"key": "value",
				},
			},
		},
		{
			name:  "parse as matrixValues with multiple keys",
			value: "key=value,key2=value2=value3",
			want: &execution.ParallelIndex{
				MatrixValues: map[string]string{
					"key":  "value",
					"key2": "value2=value3",
				},
			},
		},
		{
			name:  "parse as JSON",
			value: `{"matrixValues": {"key": "value", "key2": "value2"}}`,
			want: &execution.ParallelIndex{
				MatrixValues: map[string]string{
					"key":  "value",
					"key2": "value2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cmd.ParseParallelIndex(tt.value)
			if testutils.WantError(t, tt.wantErr, err, fmt.Sprintf("ParseParallelIndex(%v)", tt.value)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ParseParallelIndex(%v)", tt.value)
		})
	}
}
