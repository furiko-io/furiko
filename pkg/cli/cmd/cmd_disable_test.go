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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/furiko-io/furiko/pkg/cli/cmd"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
	"github.com/furiko-io/furiko/pkg/utils/errors"
)

func TestDisableCommand(t *testing.T) {
	runtimetesting.RunCommandTests(t, []runtimetesting.CommandTest{
		{
			Name: "display help",
			Args: []string{"disable", "--help"},
			Stdout: runtimetesting.Output{
				Contains: cmd.DisableExample,
			},
		},
		{
			Name:      "need an argument",
			Args:      []string{"disable"},
			WantError: assert.Error,
		},
		{
			Name:      "job config does not exist",
			Args:      []string{"run", "periodic-jobconfig"},
			WantError: errors.AssertIsNotFound(),
		},
		{
			Name:     "already disabled",
			Args:     []string{"disable", "periodic-jobconfig"},
			Fixtures: []runtime.Object{disabledJobConfig},
			Stdout: runtimetesting.Output{
				Contains: "is already disabled",
			},
		},
		{
			Name:      "job config has no schedule",
			Args:      []string{"disable", "adhoc-jobconfig"},
			Fixtures:  []runtime.Object{adhocJobConfig},
			WantError: assert.Error,
		},
		{
			Name:     "successfully disabled",
			Args:     []string{"disable", "periodic-jobconfig"},
			Fixtures: []runtime.Object{periodicJobConfig},
			Stdout: runtimetesting.Output{
				Contains: "Successfully disabled automatic scheduling",
			},
			WantActions: runtimetesting.CombinedActions{
				Furiko: runtimetesting.ActionTest{
					Actions: []runtimetesting.Action{
						runtimetesting.NewUpdateJobConfigAction(DefaultNamespace, disabledJobConfig),
					},
				},
			},
		},
	})
}
