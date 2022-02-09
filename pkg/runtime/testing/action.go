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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

	stringsutils "github.com/furiko-io/furiko/pkg/utils/strings"
)

var (
	defaultVerbs = []string{
		"create",
		"update",
		"patch",
		"delete",
	}
)

type ActionTest struct {
	// Types contains the list of verbs that should be checked.
	// Defaults to write-only verbs.
	Verbs []string

	// Actions contains a list of actions that should exist in the result.
	// It is expected that they will be in the correct order.
	Actions []ktesting.Action

	// ActionGenerators is like Actions except that generating an Action may fail
	// with an error. If specified, will take precedence over Actions.
	ActionGenerators []ActionGenerator
}

// ActionGenerator generates an Action or throws an error.
type ActionGenerator func() (ktesting.Action, error)

func (t ActionTest) GetVerbs() []string {
	if len(t.Verbs) == 0 {
		return defaultVerbs
	}
	return t.Verbs
}

func (t ActionTest) GetActions() ([]ktesting.Action, error) {
	if len(t.ActionGenerators) > 0 {
		actions := make([]ktesting.Action, 0, len(t.ActionGenerators))
		for _, gen := range t.ActionGenerators {
			action, err := gen()
			if err != nil {
				return nil, err
			}
			actions = append(actions, action)
		}
		return actions, nil
	}

	return t.Actions, nil
}

// CompareActions compares the actions we received against the ActionTest spec.
func CompareActions(t *testing.T, test ActionTest, got []ktesting.Action) {
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
			t.Errorf("saw extra action %v", gotAction)
			return
		}
		wantAction := actions[idx]
		idx++
		CompareAction(t, wantAction, gotAction)
	}

	if idx < len(actions) {
		t.Errorf("wanted %v actions, saw %v actions (verbs = %v)", len(actions), idx, test.GetVerbs())
	}
}

type ObjectGetter interface {
	GetObject() runtime.Object
}

type NameGetter interface {
	GetName() string
}

type PatchGetter interface {
	GetPatch() []byte
}

// CompareAction compares two Actions.
func CompareAction(t *testing.T, want, got ktesting.Action) {
	if !want.Matches(got.GetVerb(), got.GetResource().Resource) {
		t.Errorf("mismatched actions, want %v %v got %v %v",
			want.GetVerb(), want.GetResource(), got.GetVerb(), got.GetResource())
		return
	}

	// Compare by ObjectGetter.
	if wantObj, ok := want.(ObjectGetter); ok {
		if gotObj, ok := got.(ObjectGetter); ok {
			CompareObjects(t, wantObj, gotObj)
		}
	}

	// Compare by NameGetter.
	if wantObj, ok := want.(NameGetter); ok {
		if gotObj, ok := got.(NameGetter); ok {
			CompareNames(t, wantObj, gotObj)
		}
	}

	// Compare by PatchGetter.
	if wantObj, ok := want.(PatchGetter); ok {
		if gotObj, ok := got.(PatchGetter); ok {
			ComparePatches(t, wantObj, gotObj)
		}
	}
}

// CompareObjects compares two objects.
func CompareObjects(t *testing.T, want, got ObjectGetter) {
	wantGVK := want.GetObject().GetObjectKind().GroupVersionKind()
	gotGVK := got.GetObject().GetObjectKind().GroupVersionKind()

	if wantGVK != gotGVK {
		t.Errorf("mismatched kinds, want %v got %v", wantGVK, gotGVK)
		return
	}

	if !cmp.Equal(want.GetObject(), got.GetObject(), cmpopts.EquateEmpty()) {
		t.Errorf("mismatched objects\ndiff = %v", cmp.Diff(want.GetObject(), got.GetObject()))
	}
}

// CompareNames compares two names.
func CompareNames(t *testing.T, want, got NameGetter) {
	if want.GetName() != got.GetName() {
		t.Errorf("mismatched names, want %v got %v", want.GetName(), got.GetName())
	}
}

// ComparePatches compares two patches.
func ComparePatches(t *testing.T, want, got PatchGetter) {
	if string(want.GetPatch()) != string(got.GetPatch()) {
		t.Errorf("mismatched patches, want %v got %v", string(want.GetPatch()), string(got.GetPatch()))
	}
}
