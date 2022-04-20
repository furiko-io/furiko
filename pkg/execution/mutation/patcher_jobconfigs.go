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
	"github.com/furiko-io/furiko/pkg/runtime/webhook"
)

// JobConfigPatcher encapsulates high-level patch methods for JobConfigs.
type JobConfigPatcher struct {
	mutator *Mutator
}

func NewJobConfigPatcher(ctrlContext controllercontext.Context) *JobConfigPatcher {
	return &JobConfigPatcher{
		mutator: NewMutator(ctrlContext),
	}
}

func (p *JobConfigPatcher) Patch(operation admissionv1.Operation, oldRjc, rjc *execution.JobConfig) *webhook.Result {
	switch operation {
	case admissionv1.Create:
		return p.patchCreate(oldRjc, rjc)
	case admissionv1.Update:
		return p.patchUpdate(oldRjc, rjc)
	}

	// Unhandled operation
	return webhook.NewResult()
}

func (p *JobConfigPatcher) patchCreate(_, rjc *execution.JobConfig) *webhook.Result {
	result := webhook.NewResult()
	result.Merge(p.mutator.MutateCreateJobConfig(rjc))
	result.Merge(p.mutator.MutateJobConfig(rjc))
	return result
}

func (p *JobConfigPatcher) patchUpdate(oldRjc, rjc *execution.JobConfig) *webhook.Result {
	result := webhook.NewResult()
	result.Merge(p.mutator.MutateJobConfig(rjc))
	result.Merge(p.mutator.MutateUpdateJobConfig(oldRjc, rjc))
	return result
}
