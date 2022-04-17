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

package jobconfigcontroller_test

import (
	"testing"

	"k8s.io/client-go/tools/record"

	"github.com/furiko-io/furiko/pkg/execution/controllers/jobconfigcontroller"
	"github.com/furiko-io/furiko/pkg/execution/stores/activejobstore"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext/mock"
	"github.com/furiko-io/furiko/pkg/runtime/reconciler"
	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
)

func TestReconciler(t *testing.T) {
	test := runtimetesting.ReconcilerTest{
		ContextFunc: func(c controllercontext.Context, recorder record.EventRecorder) runtimetesting.ControllerContext {
			return jobconfigcontroller.NewContextWithRecorder(c, recorder)
		},
		ReconcilerFunc: func(c runtimetesting.ControllerContext) reconciler.Reconciler {
			return jobconfigcontroller.NewReconciler(
				c.(*jobconfigcontroller.Context),
				runtimetesting.ReconcilerDefaultConcurrency,
			)
		},
		Stores: []mock.StoreFactory{
			activejobstore.NewFactory(),
		},
	}

	test.Run(t, nil)
}
