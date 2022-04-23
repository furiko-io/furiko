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

package main

import (
	"fmt"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/furiko-io/furiko/pkg/cli/cmd"
	"github.com/furiko-io/furiko/pkg/cli/util/prompt"
)

func main() {
	ctx := ctrl.SetupSignalHandler()
	err := cmd.NewRootCommand(ctx).Execute()
	if err != nil {
		// Special handling: Do not raise interrupts as errors, but simply return the
		// exit code when signaled by SIGINT.
		if prompt.IsInterruptError(err) {
			os.Exit(130)
		}

		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
