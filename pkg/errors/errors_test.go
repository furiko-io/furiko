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

package errors_test

import (
	"testing"

	"github.com/pkg/errors"

	rerrors "github.com/furiko-io/furiko/pkg/errors"
)

func TestAdmissionRefused(t *testing.T) {
	var err error

	err = rerrors.NewAdmissionRefusedError("test message")
	if !rerrors.IsAdmissionRefused(err) {
		t.Errorf("expected IsAdmissionRefused to be true")
	} else {
		if rerr := rerrors.Error(nil); !errors.As(err, &rerr) {
			t.Errorf("expected err to be Error")
		}
		if rerrors.GetReason(err) != rerrors.ReasonAdmissionRefused {
			t.Errorf(`expected Reason to be "AdmissionRefused", got "%v"`, rerrors.GetReason(err))
		}
		if rerrors.GetMessage(err) != "test message" {
			t.Errorf(`expected Message to be "test message", got "%v"`, rerrors.GetMessage(err))
		}
	}
	if err.Error() != "AdmissionRefused - test message" {
		t.Errorf(`expected Message to be "AdmissionRefused - test message", got %v`, err.Error())
	}

	err = errors.New("some other error")
	if rerrors.IsAdmissionRefused(err) {
		t.Errorf("expected IsAdmissionRefused to be false")
	}
	if rerr := rerrors.Error(nil); errors.As(err, &rerr) {
		t.Errorf("expected err to not be Error")
	}
	if rerrors.GetReason(err) != rerrors.ReasonUnknown {
		t.Errorf(`expected Reason to be "ReasonUnknown", got "%v"`, rerrors.GetReason(err))
	}
	if rerrors.GetMessage(err) != "" {
		t.Errorf(`expected Message to be "", got "%v"`, rerrors.GetMessage(err))
	}
}
