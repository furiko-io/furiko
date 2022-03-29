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

package testing

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	testinginterface "github.com/mitchellh/go-testing-interface"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

	stringsutils "github.com/furiko-io/furiko/pkg/utils/strings"
)

type ObjectGetter interface {
	GetObject() runtime.Object
}

type NameGetter interface {
	GetName() string
}

type PatchGetter interface {
	GetPatch() []byte
}

// CompareActions compares the actions we received against the ActionTest spec.
func CompareActions(t testinginterface.T, test ActionTest, got []ktesting.Action) {
	actions, err := test.GetActions()
	if err != nil {
		t.Fatalf("cannot get actions from test: %v", err)
		return
	}

	var idx int
	for _, gotAction := range got {
		if !stringsutils.ContainsString(test.GetVerbs(), gotAction.GetVerb()) {
			continue
		}
		if idx >= len(actions) {
			t.Errorf("saw extra action: %v %v", gotAction.GetVerb(), GetFullResourceName(gotAction))
			continue
		}
		wantAction := actions[idx]
		if err := CompareAction(wantAction, gotAction); err != nil {
			t.Errorf("action %d (%v %v) did not match received action (%v %v): %v",
				idx+1, wantAction.GetVerb(), GetFullResourceName(wantAction),
				gotAction.GetVerb(), GetFullResourceName(gotAction), err)
		}
		idx++
	}

	for i := idx; i < len(actions); i++ {
		action := actions[i]
		t.Errorf("did not see action: %v %v", action.GetVerb(), GetFullResourceName(action))
	}
}

// CompareAction compares two Actions.
func CompareAction(want Action, got ktesting.Action) error {
	if !want.Matches(got.GetVerb(), got.GetResource().Resource) {
		return fmt.Errorf("mismatched actions, want %v %v got %v %v",
			want.GetVerb(), want.GetResource(), got.GetVerb(), got.GetResource())
	}

	if want.GetNamespace() != got.GetNamespace() {
		return fmt.Errorf("mismatched namespace, want %v got %v", want.GetNamespace(), got.GetNamespace())
	}

	// Compare by ObjectGetter.
	if wantObj, ok := want.Action.(ObjectGetter); ok && !want.IgnoreObject {
		if gotObj, ok := got.(ObjectGetter); ok {
			if err := CompareObjects(wantObj, gotObj); err != nil {
				return err
			}
		}
	}

	// Compare by NameGetter.
	if wantObj, ok := want.Action.(NameGetter); ok {
		if gotObj, ok := got.(NameGetter); ok {
			if err := CompareNames(wantObj, gotObj); err != nil {
				return err
			}
		}
	}

	// Compare by PatchGetter.
	if wantObj, ok := want.Action.(PatchGetter); ok {
		if gotObj, ok := got.(PatchGetter); ok {
			if err := ComparePatches(wantObj, gotObj); err != nil {
				return err
			}
		}
	}

	return nil
}

// CompareObjects compares two objects.
func CompareObjects(want, got ObjectGetter) error {
	wantGVK := want.GetObject().GetObjectKind().GroupVersionKind()
	gotGVK := got.GetObject().GetObjectKind().GroupVersionKind()

	if wantGVK != gotGVK {
		return fmt.Errorf("mismatched kinds, want %v got %v", wantGVK, gotGVK)
	}

	if !cmp.Equal(want.GetObject(), got.GetObject(), cmpopts.EquateEmpty()) {
		return fmt.Errorf("mismatched objects\ndiff = %v", cmp.Diff(want.GetObject(), got.GetObject()))
	}

	return nil
}

// CompareNames compares two names.
func CompareNames(want, got NameGetter) error {
	if want.GetName() != got.GetName() {
		return fmt.Errorf("mismatched names, want %v got %v", want.GetName(), got.GetName())
	}
	return nil
}

// ComparePatches compares two patches.
func ComparePatches(want, got PatchGetter) error {
	if string(want.GetPatch()) != string(got.GetPatch()) {
		return fmt.Errorf("mismatched patches, want %v got %v", string(want.GetPatch()), string(got.GetPatch()))
	}
	return nil
}
