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

package validation

import (
	"fmt"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ToInternalErrorList converts an error into an ErrorList with a single InternalError.
func ToInternalErrorList(fldPath *field.Path, err error) field.ErrorList {
	return field.ErrorList{
		field.InternalError(fldPath, err),
	}
}

// ValidateImmutableField validates the new value and the old value are deeply
// equal and returns an error with a custom message.
func ValidateImmutableField(newVal, oldVal interface{}, fldPath *field.Path, msg string) field.ErrorList {
	allErrs := field.ErrorList{}
	if !apiequality.Semantic.DeepEqual(oldVal, newVal) {
		allErrs = append(allErrs, field.Invalid(fldPath, newVal, msg))
	}
	return allErrs
}

// ValidateMaxLength validates the value to enforce a maximum length.
func ValidateMaxLength(val string, maxLen int, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(val) > maxLen {
		allErrs = append(allErrs, field.Invalid(fldPath, val, fmt.Sprintf("cannot be more than %v characters", maxLen)))
	}
	return allErrs
}
