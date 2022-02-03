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

type ObjectMeta interface {
	GetAnnotations() map[string]string
	SetAnnotations(annotations map[string]string)
}

// SetAnnotation updates the object's annotations, setting the annotation given
// by key to be value. The Annotations map is updated in-place, if needed please
// DeepCopy before calling this function.
func SetAnnotation(object ObjectMeta, key, value string) {
	annotations := object.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	object.SetAnnotations(annotations)
}
