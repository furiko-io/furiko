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

package printer_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/furiko-io/furiko/pkg/cli/printer"
)

const (
	mockPod1YAML = `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: pod-1
  namespace: test
spec:
  containers: null
status: {}`
	mockPodListYAML = `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: pod-1
  namespace: test
spec:
  containers: null
status: {}
---
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: pod-2
  namespace: test
spec:
  containers: null
status: {}`
	mockPod1JSON = `{
    "kind": "Pod",
    "apiVersion": "v1",
    "metadata": {
        "name": "pod-1",
        "namespace": "test",
        "creationTimestamp": null
    },
    "spec": {
        "containers": null
    },
    "status": {}
}`
	mockPodListJSON = `{
    "kind": "Pod",
    "apiVersion": "v1",
    "metadata": {
        "name": "pod-1",
        "namespace": "test",
        "creationTimestamp": null
    },
    "spec": {
        "containers": null
    },
    "status": {}
}
{
    "kind": "Pod",
    "apiVersion": "v1",
    "metadata": {
        "name": "pod-2",
        "namespace": "test",
        "creationTimestamp": null
    },
    "spec": {
        "containers": null
    },
    "status": {}
}`
	mockPod1Name    = `pod/pod-1`
	mockPodListName = `pod/pod-1
pod/pod-2`
)

var (
	gvkPod = corev1.SchemeGroupVersion.WithKind("Pod")

	managedFields = []metav1.ManagedFieldsEntry{
		{Manager: "kubectl", Operation: metav1.ManagedFieldsOperationApply},
	}

	mockPod1 = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "pod-1",
		},
	}

	mockPod1WithManagedFields = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:     "test",
			Name:          "pod-1",
			ManagedFields: managedFields,
		},
	}

	mockPod2 = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "pod-2",
		},
	}

	mockPod2WithManagedFields = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:     "test",
			Name:          "pod-2",
			ManagedFields: managedFields,
		},
	}
)

func TestPrintObject(t *testing.T) {
	tests := []struct {
		name    string
		gvk     schema.GroupVersionKind
		output  printer.OutputFormat
		object  printer.Object
		wantOut string
		wantErr bool
	}{
		{
			name:    "unknown format",
			gvk:     gvkPod,
			output:  "invalid",
			object:  mockPod1,
			wantErr: true,
		},
		{
			name:    "print pod yaml",
			gvk:     gvkPod,
			output:  printer.OutputFormatYAML,
			object:  mockPod1,
			wantOut: mockPod1YAML,
		},
		{
			name:    "print pod yaml without managed fields",
			gvk:     gvkPod,
			output:  printer.OutputFormatYAML,
			object:  mockPod1WithManagedFields,
			wantOut: mockPod1YAML,
		},
		{
			name:    "print pod json",
			gvk:     gvkPod,
			output:  printer.OutputFormatJSON,
			object:  mockPod1,
			wantOut: mockPod1JSON,
		},
		{
			name:    "print pod json without managed fields",
			gvk:     gvkPod,
			output:  printer.OutputFormatJSON,
			object:  mockPod1WithManagedFields,
			wantOut: mockPod1JSON,
		},
		{
			name:    "print pod name",
			gvk:     gvkPod,
			output:  printer.OutputFormatName,
			object:  mockPod1,
			wantOut: mockPod1Name,
		},
		{
			name:    "print pod name without managed fields",
			gvk:     gvkPod,
			output:  printer.OutputFormatName,
			object:  mockPod1WithManagedFields,
			wantOut: mockPod1Name,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			err := printer.PrintObject(tt.gvk, tt.output, out, tt.object)
			if (err != nil) != tt.wantErr {
				t.Errorf("PrintObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotOut := out.String()

			// We always expect a trailing newline, we don't add it in the test fixtures for cleanliness
			wantOut := tt.wantOut + "\n"

			// In case of the YAML printer, strip the leading "---" if any.
			gotOut = strings.TrimPrefix(gotOut, "---\n")

			if gotOut != wantOut {
				t.Errorf("PrintObject() not equal\ndiff = %v", cmp.Diff(wantOut, gotOut))
			}
		})
	}
}

func TestPrintObjects(t *testing.T) {
	tests := []struct {
		name    string
		gvk     schema.GroupVersionKind
		output  printer.OutputFormat
		objects []printer.Object
		wantOut string
		wantErr bool
	}{
		{
			name:   "unknown format",
			gvk:    gvkPod,
			output: "invalid",
			objects: []printer.Object{
				mockPod1,
			},
			wantErr: true,
		},
		{
			name:   "print single pod yaml",
			gvk:    gvkPod,
			output: printer.OutputFormatYAML,
			objects: []printer.Object{
				mockPod1,
			},
			wantOut: mockPod1YAML,
		},
		{
			name:   "print single pod yaml without managed fields",
			gvk:    gvkPod,
			output: printer.OutputFormatYAML,
			objects: []printer.Object{
				mockPod1WithManagedFields,
			},
			wantOut: mockPod1YAML,
		},
		{
			name:   "print single pod json",
			gvk:    gvkPod,
			output: printer.OutputFormatJSON,
			objects: []printer.Object{
				mockPod1,
			},
			wantOut: mockPod1JSON,
		},
		{
			name:   "print single pod json without managed fields",
			gvk:    gvkPod,
			output: printer.OutputFormatJSON,
			objects: []printer.Object{
				mockPod1WithManagedFields,
			},
			wantOut: mockPod1JSON,
		},
		{
			name:   "print single pod name",
			gvk:    gvkPod,
			output: printer.OutputFormatName,
			objects: []printer.Object{
				mockPod1,
			},
			wantOut: mockPod1Name,
		},
		{
			name:   "print single pod name without managed fields",
			gvk:    gvkPod,
			output: printer.OutputFormatName,
			objects: []printer.Object{
				mockPod1WithManagedFields,
			},
			wantOut: mockPod1Name,
		},
		{
			name:   "print pod list yaml",
			gvk:    gvkPod,
			output: printer.OutputFormatYAML,
			objects: []printer.Object{
				mockPod1,
				mockPod2,
			},
			wantOut: mockPodListYAML,
		},
		{
			name:   "print pod list yaml without managed fields",
			gvk:    gvkPod,
			output: printer.OutputFormatYAML,
			objects: []printer.Object{
				mockPod1WithManagedFields,
				mockPod2WithManagedFields,
			},
			wantOut: mockPodListYAML,
		},
		{
			name:   "print pod list json",
			gvk:    gvkPod,
			output: printer.OutputFormatJSON,
			objects: []printer.Object{
				mockPod1,
				mockPod2,
			},
			wantOut: mockPodListJSON,
		},
		{
			name:   "print pod list json without managed fields",
			gvk:    gvkPod,
			output: printer.OutputFormatJSON,
			objects: []printer.Object{
				mockPod1WithManagedFields,
				mockPod2WithManagedFields,
			},
			wantOut: mockPodListJSON,
		},
		{
			name:   "print pod name json",
			gvk:    gvkPod,
			output: printer.OutputFormatName,
			objects: []printer.Object{
				mockPod1,
				mockPod2,
			},
			wantOut: mockPodListName,
		},
		{
			name:   "print pod name json without managed fields",
			gvk:    gvkPod,
			output: printer.OutputFormatName,
			objects: []printer.Object{
				mockPod1WithManagedFields,
				mockPod2WithManagedFields,
			},
			wantOut: mockPodListName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			err := printer.PrintObjects(tt.gvk, tt.output, out, tt.objects)
			if (err != nil) != tt.wantErr {
				t.Errorf("PrintObjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotOut := out.String()

			// We always expect a trailing newline, we don't add it in the test fixtures for cleanliness
			wantOut := tt.wantOut + "\n"

			// In case of the YAML printer, strip the leading "---" if any.
			gotOut = strings.TrimPrefix(gotOut, "---\n")

			if gotOut != wantOut {
				t.Errorf("PrintObjects() not equal\ndiff = %v", cmp.Diff(wantOut, gotOut))
			}
		})
	}
}
