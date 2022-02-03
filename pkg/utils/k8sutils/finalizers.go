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

package k8sutils

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// IsDeleting returns true if the object is pending deletion.
func IsDeleting(object metav1.Object) bool {
	return object.GetDeletionTimestamp() != nil && !object.GetDeletionTimestamp().IsZero()
}

// IsPendingFinalizer returns true if the object is pending deletion and contains the given finalizer.
// Use this to determine when to remove the finalizer.
func IsPendingFinalizer(object metav1.Object, finalizer string) bool {
	return IsDeleting(object) && ContainsFinalizer(object.GetFinalizers(), finalizer)
}

// FilterFinalizers filters a list of finalizers with some filter function.
func FilterFinalizers(finalizers []string, filter func(f string) bool) []string {
	filtered := make([]string, 0, len(finalizers))
	for _, f := range finalizers {
		if filter(f) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// ContainsFinalizer returns true if the given finalizer is in the list.
func ContainsFinalizer(finalizers []string, finalizer string) bool {
	for _, f := range finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

// MergeFinalizers merges the list of finalizers, removing duplicates if any.
func MergeFinalizers(finalizers1, finalizers2 []string) []string {
	newFinalizers := make([]string, 0, len(finalizers1)+len(finalizers2))
	m := sets.NewString(finalizers1...)
	newFinalizers = append(newFinalizers, finalizers1...)
	for _, f := range finalizers2 {
		if !m.Has(f) {
			newFinalizers = append(newFinalizers, f)
		}
	}
	return newFinalizers
}

// RemoveFinalizer removes the given finalizer from the list of finalizers.
func RemoveFinalizer(finalizers []string, finalizer string) []string {
	newFinalizers := make([]string, 0, len(finalizers))
	for _, f := range finalizers {
		if f != finalizer {
			newFinalizers = append(newFinalizers, f)
		}
	}
	return newFinalizers
}

// IsFinalFinalizer returns whether the list of finalizers does not contain other finalizers with the given prefix.
// Pass a prefix to filter finalizers, to prevent blocking on other finalizers, e.g. foregroundDeletion.
func IsFinalFinalizer(prefix string, finalizers []string, finalizer string) bool {
	// Filter finalizers with prefix.
	filtered := FilterFinalizers(finalizers, func(f string) bool {
		return strings.HasPrefix(f, prefix)
	})

	return len(filtered) == 1 && ContainsFinalizer(filtered, finalizer)
}
