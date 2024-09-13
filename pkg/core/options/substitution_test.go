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

package options_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/furiko-io/furiko/pkg/core/options"
)

const (
	jobID  = "my-job-id"
	taskID = "my-job-id.1"
)

func TestSubstituteVariables(t *testing.T) {
	type args struct {
		target string
		submap map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no substitutions",
			args: args{
				target: "echo ${job.id}",
				submap: map[string]string{},
			},
			want: "echo ${job.id}",
		},
		{
			name: "nil submap",
			args: args{
				target: "echo ${job.id}",
				submap: nil,
			},
			want: "echo ${job.id}",
		},
		{
			name: "case-sensitive",
			args: args{
				target: "echo ${job.ID}",
				submap: map[string]string{
					"job.id": jobID,
				},
			},
			want: "echo ${job.ID}",
		},
		{
			name: "correct substitution",
			args: args{
				target: `echo "${job.id}"`,
				submap: map[string]string{
					"job.id": jobID,
				},
			},
			want: fmt.Sprintf(`echo "%v"`, jobID),
		},
		{
			name: "can escape",
			args: args{
				target: `echo "\$\{job.id\}"`,
				submap: map[string]string{
					"job.id": jobID,
				},
			},
			want: `echo "\$\{job.id\}"`,
		},
		{
			name: "no substitution",
			args: args{
				target: "echo Hello",
				submap: map[string]string{
					"job.id": jobID,
				},
			},
			want: "echo Hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := options.SubstituteVariables(tt.args.target, tt.args.submap); got != tt.want {
				t.Errorf("SubstituteVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubstituteEmptyStringForPrefixes(t *testing.T) {
	type args struct {
		target   string
		prefixes []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty string",
			args: args{
				target:   "",
				prefixes: []string{"job."},
			},
			want: "",
		},
		{
			name: "no prefixes",
			args: args{
				target:   "echo ${job.id}",
				prefixes: nil,
			},
			want: "echo ${job.id}",
		},
		{
			name: "only specified prefixes",
			args: args{
				target:   "echo ${job.id} ${task.execid}",
				prefixes: []string{"job."},
			},
			want: "echo  ${task.execid}",
		},
		{
			name: "don't substitute bash variables",
			args: args{
				target:   "echo ${job} ${job.id}",
				prefixes: []string{"job."},
			},
			want: "echo ${job} ",
		},
		{
			name: "support prefix without trailing dot",
			args: args{
				target:   "echo ${job} ${job.id}",
				prefixes: []string{"job"},
			},
			want: "echo ${job} ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := options.SubstituteEmptyStringForPrefixes(tt.args.target, tt.args.prefixes); got != tt.want {
				t.Errorf("SubstituteEmptyStringForPrefixes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubstituteVariableMaps(t *testing.T) {
	type args struct {
		target   string
		submaps  []map[string]string
		prefixes []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty string",
			args: args{
				target: "",
				submaps: []map[string]string{
					{
						"job.id": jobID,
					},
				},
				prefixes: []string{"job"},
			},
			want: "",
		},
		{
			name: "replace job variables, replace empty string, don't touch task variables",
			args: args{
				target: `echo Job ID: "${job.id}"; echo Job Abc: "${job.abc}"; echo Task ID: "${task.id}";`,
				submaps: []map[string]string{
					{
						"job.id": jobID,
					},
				},
				prefixes: []string{"job"},
			},
			want: fmt.Sprintf(`echo Job ID: "%v"; echo Job Abc: ""; echo Task ID: "${task.id}";`, jobID),
		},
		{
			name: "replace all job and task variables",
			args: args{
				target: `echo Job ID: "${job.id}"; echo Job Abc: "${job.abc}"; echo Task ID: "${task.id}";`,
				submaps: []map[string]string{
					{
						"job.id": jobID,
					},
				},
				prefixes: []string{"job", "task"},
			},
			want: fmt.Sprintf(`echo Job ID: "%v"; echo Job Abc: ""; echo Task ID: "";`, jobID),
		},
		{
			name: "substitute job and task variables",
			args: args{
				target: `echo Job ID: "${job.id}"; echo Job Abc: "${job.abc}"; echo Task ID: "${task.id}";`,
				submaps: []map[string]string{
					{
						"job.id":  jobID,
						"task.id": taskID,
					},
				},
				prefixes: []string{"job", "task"},
			},
			want: fmt.Sprintf(`echo Job ID: "%v"; echo Job Abc: ""; echo Task ID: "%v";`, jobID, taskID),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := options.SubstituteVariableMaps(tt.args.target, tt.args.submaps, tt.args.prefixes)
			if got != tt.want {
				t.Errorf("SubstituteVariableMaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeSubstitutions(t *testing.T) {
	tests := []struct {
		name      string
		paramMaps []map[string]string
		want      map[string]string
	}{
		{
			name:      "no maps",
			paramMaps: nil,
			want:      map[string]string{},
		},
		{
			name: "nil maps",
			paramMaps: []map[string]string{
				nil,
				{
					"option.abc": "abc",
				},
			},
			want: map[string]string{
				"option.abc": "abc",
			},
		},
		{
			name: "override",
			paramMaps: []map[string]string{
				{
					"option.abc": "abc",
					"option.def": "def",
				},
				{
					"option.abc": "abc2",
					"option.ghi": "ghi",
				},
			},
			want: map[string]string{
				"option.abc": "abc2",
				"option.def": "def",
				"option.ghi": "ghi",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := options.MergeSubstitutions(tt.paramMaps...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
