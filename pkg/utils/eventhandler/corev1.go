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

package eventhandler

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Corev1Pod casts obj into *corev1.Pod.
func Corev1Pod(obj interface{}) (*corev1.Pod, error) {
	switch t := obj.(type) {
	case *corev1.Pod:
		return t, nil
	case cache.DeletedFinalStateUnknown:
		obj, ok := t.Obj.(*corev1.Pod)
		if ok {
			return obj, nil
		}
	}

	return nil, NewUnexpectedTypeError(&corev1.Pod{}, obj)
}

// Corev1ConfigMap casts obj into *corev1.ConfigMap.
func Corev1ConfigMap(obj interface{}) (*corev1.ConfigMap, error) {
	switch t := obj.(type) {
	case *corev1.ConfigMap:
		return t, nil
	case cache.DeletedFinalStateUnknown:
		obj, ok := t.Obj.(*corev1.ConfigMap)
		if ok {
			return obj, nil
		}
	}

	return nil, NewUnexpectedTypeError(&corev1.ConfigMap{}, obj)
}

// Corev1Secret casts obj into *corev1.Secret.
func Corev1Secret(obj interface{}) (*corev1.Secret, error) {
	switch t := obj.(type) {
	case *corev1.Secret:
		return t, nil
	case cache.DeletedFinalStateUnknown:
		obj, ok := t.Obj.(*corev1.Secret)
		if ok {
			return obj, nil
		}
	}

	return nil, NewUnexpectedTypeError(&corev1.Secret{}, obj)
}
