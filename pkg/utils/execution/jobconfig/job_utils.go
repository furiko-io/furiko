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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	lister "github.com/furiko-io/furiko/pkg/generated/listers/execution/v1alpha1"
)

// GetLastScheduleTime returns the latest schedule time in a list of all Jobs.
// If the list is empty, or none of the jobs were scheduled jobs, will return nil.
func GetLastScheduleTime(jobs []*execution.Job) *metav1.Time {
	var lastScheduleTime metav1.Time
	for _, rj := range jobs {
		rj := rj
		if t := GetLabelScheduleTime(rj); t != nil && lastScheduleTime.Before(t) {
			lastScheduleTime = *t
		}
	}
	if lastScheduleTime.IsZero() {
		return nil
	}
	return &lastScheduleTime
}

// GetLabelScheduleTime returns the cron schedule time for a Job from its annotation.
// If no annotation exists or is not a valid Unix timestamp, nil will be returned.
func GetLabelScheduleTime(rj *execution.Job) *metav1.Time {
	if val, ok := rj.Annotations[AnnotationKeyScheduleTime]; ok {
		if ts, err := strconv.Atoi(val); err == nil {
			mt := metav1.NewTime(time.Unix(int64(ts), 0))
			return &mt
		}
	}
	return nil
}

// LookupJobOwner looks up the JobConfig for a Job using the ownerReferences
// field if it exists. Otherwise, both a nil JobConfig and a nil error is
// returned if there are no owner references.
//
// All Jobs with a JobConfig parent (specified via ownerReferences) are expected
// to have a special label (see LabelKeyJobConfigUID), which is used for
// internal selectors and cannot be updated.
func LookupJobOwner(rj *execution.Job, lister lister.JobConfigNamespaceLister) (*execution.JobConfig, error) {
	rjc, errs := ValidateLookupJobOwner(rj, lister)
	if err := errs.ToAggregate(); err != nil {
		return nil, err
	}
	return rjc, nil
}

// ValidateLookupJobOwner looks up the owner JobConfig for a Job and returns
// validation errors if any.
//
// All Jobs with a JobConfig parent (specified via ownerReferences) are expected
// to have a special label (see LabelKeyJobConfigUID), which is used for
// internal selectors and cannot be updated. This method also performs sanity
// checks to ensure that the returned JobConfig matches the UID fields in the
// Job.
//
// May return a nil JobConfig if there is no owner reference defined.
func ValidateLookupJobOwner(
	rj *execution.Job,
	lister lister.JobConfigNamespaceLister,
) (*execution.JobConfig, field.ErrorList) {
	ref := metav1.GetControllerOf(rj)

	errorList := field.ErrorList{}
	fldPath := field.NewPath("metadata")
	ownerReferencesPath := fldPath.Child("ownerReferences")

	// Did not find any ownerReference which is the controller for this Job.
	if ref == nil || ref.Kind != execution.KindJobConfig {
		return nil, nil
	}

	// Use correct index for ownerReferences.
	var index int
	for i, reference := range rj.OwnerReferences {
		if reference == *ref {
			index = i
			break
		}
	}
	ownerReferencePath := ownerReferencesPath.Index(index)

	// Look up JobConfig by name.
	rjc, err := lister.Get(ref.Name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			errorList = append(errorList, field.NotFound(ownerReferencePath, ref.Name))
		} else {
			err := errors.Wrapf(err, "cannot get jobconfig")
			errorList = append(errorList, field.InternalError(ownerReferencePath, err))
		}
		return nil, errorList
	}

	// The ownerReference may have a mismatched UID, which could happen if the
	// resource was deleted and recreated with the same name.
	if rjc.UID != ref.UID {
		fldPath := ownerReferencePath.Key("uid")
		errorList = append(errorList, field.Duplicate(fldPath, rjc.UID))
		return nil, errorList
	}

	// Ensure that UID label exists and matches.
	if label := rj.Labels[LabelKeyJobConfigUID]; label != string(rjc.UID) {
		fldPath := fldPath.Child("labels").Key(LabelKeyJobConfigUID)
		errorList = append(errorList, field.Required(fldPath,
			"label must be specified if ownerReferences is specified"))
		return nil, errorList
	}

	return rjc, nil
}
