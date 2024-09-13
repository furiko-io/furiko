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

package meta_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/furiko-io/furiko/pkg/utils/meta"
)

type object struct {
	annotations map[string]string
}

func (o *object) GetAnnotations() map[string]string {
	return o.annotations
}

func (o *object) SetAnnotations(annotations map[string]string) {
	o.annotations = annotations
}

func TestSetAnnotation(t *testing.T) {
	tests := []struct {
		name   string
		object meta.Annotatable
		key    string
		value  string
		want   map[string]string
	}{
		{
			name:   "nil map",
			object: &object{annotations: nil},
			key:    "key",
			value:  "value",
			want:   map[string]string{"key": "value"},
		},
		{
			name:   "empty map",
			object: &object{annotations: map[string]string{}},
			key:    "key",
			value:  "value",
			want:   map[string]string{"key": "value"},
		},
		{
			name: "map with another key",
			object: &object{annotations: map[string]string{
				"anotherkey": "anothervalue",
			}},
			key:   "key",
			value: "value",
			want: map[string]string{
				"anotherkey": "anothervalue",
				"key":        "value",
			},
		},
		{
			name: "map with existing key",
			object: &object{annotations: map[string]string{
				"anotherkey": "anothervalue",
				"key":        "oldvalue",
			}},
			key:   "key",
			value: "value",
			want: map[string]string{
				"anotherkey": "anothervalue",
				"key":        "value",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta.SetAnnotation(tt.object, tt.key, tt.value)
			got := tt.object.GetAnnotations()
			if !cmp.Equal(got, tt.want) {
				t.Errorf("annotations not equal after SetAnnotation():\ndiff = %v", cmp.Diff(tt.want, got))
			}
		})
	}
}
