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

package mutation

import (
	admissionv1 "k8s.io/api/admission/v1"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/utils/webhook"
)

// JobPatcher encapsulates high-level patch methods for Jobs.
type JobPatcher struct {
	mutator *Mutator
}

func NewJobPatcher(ctrlContext controllercontext.Context) *JobPatcher {
	return &JobPatcher{
		mutator: NewMutator(ctrlContext),
	}
}

func (p *JobPatcher) Patch(operation admissionv1.Operation, oldRj, rj *execution.Job) *webhook.Result {
	switch operation {
	case admissionv1.Create:
		return p.patchCreate(rj)
	case admissionv1.Update:
		return p.patchUpdate(oldRj, rj)
	}

	// Unhandled operation
	return webhook.NewResult()
}

func (p *JobPatcher) patchCreate(rj *execution.Job) *webhook.Result {
	result := webhook.NewResult()
	result.Merge(p.mutator.MutateCreateJob(rj))
	result.Merge(p.mutator.MutateJob(rj))
	return result
}

func (p *JobPatcher) patchUpdate(oldRj, rj *execution.Job) *webhook.Result {
	result := webhook.NewResult()
	result.Merge(p.mutator.MutateJob(rj))
	return result
}
