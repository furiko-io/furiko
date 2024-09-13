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

package testing_test

import (
	"testing"

	testinginterface "github.com/mitchellh/go-testing-interface"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"

	runtimetesting "github.com/furiko-io/furiko/pkg/runtime/testing"
)

const (
	testNamespace    = "test"
	stagingNamespace = "staging"
)

var (
	resourcePod       = runtimetesting.NewGroupVersionResource("core", "v1", "pods")
	resourceConfigMap = runtimetesting.NewGroupVersionResource("core", "v1", "configmaps")

	fakePod1 = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "pod1",
		},
	}

	fakePod2 = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "pod2",
		},
	}

	fakeConfigMap = &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "cm1",
		},
	}

	patchString = `[{"op":"replace","path": "/metadata/name","value":"name"}]`
)

func TestCompareAction(t *testing.T) {
	tests := []struct {
		name    string
		want    runtimetesting.Action
		got     ktesting.Action
		wantErr bool
	}{
		{
			name:    "different verb",
			want:    runtimetesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
			got:     ktesting.NewUpdateAction(resourcePod, testNamespace, fakePod1),
			wantErr: true,
		},
		{
			name:    "different resource",
			want:    runtimetesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
			got:     ktesting.NewCreateAction(resourceConfigMap, testNamespace, fakePod1),
			wantErr: true,
		},
		{
			name:    "different namespace",
			want:    runtimetesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
			got:     ktesting.NewCreateAction(resourcePod, stagingNamespace, fakePod1),
			wantErr: true,
		},
		{
			name:    "object kind differs",
			want:    runtimetesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
			got:     ktesting.NewCreateAction(resourcePod, testNamespace, fakeConfigMap),
			wantErr: true,
		},
		{
			name:    "object differs",
			want:    runtimetesting.NewUpdateAction(resourcePod, testNamespace, fakePod1),
			got:     ktesting.NewUpdateAction(resourcePod, testNamespace, fakePod2),
			wantErr: true,
		},
		{
			name: "IgnoreObject with different object",
			want: func() runtimetesting.Action {
				action := runtimetesting.NewUpdateAction(resourcePod, testNamespace, fakePod1)
				action.IgnoreObject = true
				return action
			}(),
			got: ktesting.NewUpdateAction(resourcePod, testNamespace, fakePod2),
		},
		{
			name: "IgnoreObject with same object",
			want: func() runtimetesting.Action {
				action := runtimetesting.NewUpdateAction(resourcePod, testNamespace, fakePod1)
				action.IgnoreObject = true
				return action
			}(),
			got: ktesting.NewUpdateAction(resourcePod, testNamespace, fakePod1),
		},
		{
			name: "object matches",
			want: runtimetesting.NewUpdateAction(resourcePod, testNamespace, fakePod1),
			got:  ktesting.NewUpdateAction(resourcePod, testNamespace, fakePod1),
		},
		{
			name: "subresource object matches",
			want: runtimetesting.NewUpdateStatusAction(resourcePod, testNamespace, fakePod1),
			got:  ktesting.NewUpdateSubresourceAction(resourcePod, "status", testNamespace, fakePod1),
		},
		{
			name:    "delete name differs",
			want:    runtimetesting.NewDeleteAction(resourcePod, testNamespace, "pod1"),
			got:     ktesting.NewDeleteAction(resourcePod, testNamespace, "pod2"),
			wantErr: true,
		},
		{
			name: "delete name matches",
			want: runtimetesting.NewDeleteAction(resourcePod, testNamespace, "pod1"),
			got:  ktesting.NewDeleteAction(resourcePod, testNamespace, "pod1"),
		},
		{
			name:    "patch differs",
			want:    runtimetesting.NewPatchAction(resourcePod, testNamespace, "pod1", types.JSONPatchType, []byte(patchString)),
			got:     ktesting.NewPatchAction(resourcePod, testNamespace, "pod1", types.JSONPatchType, []byte(`[]`)),
			wantErr: true,
		},
		{
			name:    "patch name differs",
			want:    runtimetesting.NewPatchAction(resourcePod, testNamespace, "pod1", types.JSONPatchType, []byte(patchString)),
			got:     ktesting.NewPatchAction(resourcePod, testNamespace, "pod2", types.JSONPatchType, []byte(patchString)),
			wantErr: true,
		},
		{
			name: "patch matches",
			want: runtimetesting.NewPatchAction(resourcePod, testNamespace, "pod1", types.JSONPatchType, []byte(patchString)),
			got:  ktesting.NewPatchAction(resourcePod, testNamespace, "pod1", types.JSONPatchType, []byte(patchString)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runtimetesting.CompareAction(tt.want, tt.got); (err != nil) != tt.wantErr {
				t.Errorf("CompareAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompareActions(t *testing.T) {
	tests := []struct {
		name    string
		test    runtimetesting.ActionTest
		actions []ktesting.Action
		wantErr bool
	}{
		{
			name: "action generator error",
			test: runtimetesting.ActionTest{
				ActionGenerators: []runtimetesting.ActionGenerator{
					func() (runtimetesting.Action, error) {
						return runtimetesting.Action{}, errors.New("error")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "did not see action",
			test: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
				},
			},
			wantErr: true,
		},
		{
			name: "saw extra action",
			actions: []ktesting.Action{
				runtimetesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
			},
			wantErr: true,
		},
		{
			name: "saw mismatched action verb",
			test: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
				},
			},
			actions: []ktesting.Action{
				ktesting.NewUpdateAction(resourcePod, testNamespace, fakePod1),
			},
			wantErr: true,
		},
		{
			name: "saw mismatched action subresource",
			test: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewUpdateStatusAction(resourcePod, testNamespace, fakePod1),
				},
			},
			actions: []ktesting.Action{
				ktesting.NewUpdateSubresourceAction(resourcePod, "status", testNamespace, fakePod2),
			},
			wantErr: true,
		},
		{
			name: "saw matched action",
			test: runtimetesting.ActionTest{
				Actions: []runtimetesting.Action{
					runtimetesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
				},
			},
			actions: []ktesting.Action{
				ktesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
			},
		},
		{
			name: "saw matched action using generator",
			test: runtimetesting.ActionTest{
				ActionGenerators: []runtimetesting.ActionGenerator{
					func() (runtimetesting.Action, error) {
						return runtimetesting.NewCreateAction(resourcePod, testNamespace, fakePod1), nil
					},
				},
			},
			actions: []ktesting.Action{
				ktesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
			},
		},
		{
			name: "skip some verbs",
			test: runtimetesting.ActionTest{
				Verbs: []string{"update"},
			},
			actions: []ktesting.Action{
				ktesting.NewCreateAction(resourcePod, testNamespace, fakePod1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newT := &testinginterface.RuntimeT{}
			var gotErr bool
			func() {
				defer func() {
					if r := recover(); r != nil {
						gotErr = true
					}
				}()
				runtimetesting.CompareActions(newT, tt.test, tt.actions)
			}()
			if gotErr == false {
				gotErr = newT.Failed()
			}
			if gotErr != tt.wantErr {
				t.Errorf("CompareActions() got error = %v, wantErr %v", gotErr, tt.wantErr)
			}
		})
	}
}
