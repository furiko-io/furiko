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

package jobconfig

import (
	"strconv"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	executiongroup "github.com/furiko-io/furiko/apis/execution"
	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/core/options"
)

// NewJobFromJobConfig returns a Job from a JobConfig.
func NewJobFromJobConfig(
	jobConfig *execution.JobConfig,
	jobType execution.JobType,
	createTime time.Time,
) (*execution.Job, error) {
	// Generate a unique Job name.
	jobName := GenerateName(jobConfig.GetName(), createTime)

	// Make variables.
	variables, err := makeSubstitutions(jobConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create template variables")
	}

	// Initialise Job
	job := &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   jobConfig.Namespace,
			Labels:      makeLabels(jobConfig),
			Annotations: makeAnnotations(jobConfig, jobType, createTime),
			Finalizers: []string{
				// Add finalizer so that we can synchronize deletion of dependents in the
				// JobController.
				executiongroup.DeleteDependentsFinalizer,
			},
		},
		Spec: execution.JobSpec{
			Template:      jobConfig.Spec.Template.Spec.DeepCopy(),
			Type:          jobType,
			Substitutions: variables,
		},
	}

	// Add OwnerReference back to Job
	controllerRef := metav1.NewControllerRef(jobConfig, execution.GVKJobConfig)
	job.OwnerReferences = append(job.OwnerReferences, *controllerRef)

	return job, nil
}

func makeSubstitutions(rjc *execution.JobConfig) (map[string]string, error) {
	// Evaluate all options' defaults.
	defaultSubs, err := options.MakeDefaultOptions(rjc.Spec.Option)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot evaluate default option values")
	}
	return defaultSubs, nil
}

func makeLabels(rjc *execution.JobConfig) labels.Set {
	template := rjc.Spec.Template
	desiredLabels := make(labels.Set, len(template.Labels))
	for k, v := range template.Labels {
		desiredLabels[k] = v
	}

	// Append additional labels.
	additionalLabels := map[string]string{
		LabelKeyJobConfigUID: string(rjc.GetUID()),
	}
	for k, v := range additionalLabels {
		desiredLabels[k] = v
	}

	return desiredLabels
}

func makeAnnotations(rjc *execution.JobConfig, jobType execution.JobType, createTime time.Time) labels.Set {
	template := rjc.Spec.Template
	desiredAnnotations := make(labels.Set, len(template.Annotations))
	for k, v := range template.Annotations {
		desiredAnnotations[k] = v
	}

	additionalAnnotations := map[string]string{}

	// Add schedule time label only if it was scheduled.
	if jobType == execution.JobTypeScheduled {
		additionalAnnotations[AnnotationKeyScheduleTime] = strconv.Itoa(int(createTime.Unix()))
	}

	// Append additional annotations.
	for k, v := range additionalAnnotations {
		desiredAnnotations[k] = v
	}

	return desiredAnnotations
}
