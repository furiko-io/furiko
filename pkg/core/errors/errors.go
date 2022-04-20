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

package errors

import (
	"errors"
	"fmt"
)

// Error encapsulates a semantic meaning to an error, allowing us to propagate additional error context
// between scopes easily.
type internalError struct {
	Reason  Reason
	Message string
}

func (e *internalError) Error() string {
	return fmt.Sprintf("%v - %v", e.GetReason(), e.GetMessage())
}

func (e *internalError) GetReason() Reason {
	return e.Reason
}

func (e *internalError) GetMessage() string {
	return e.Message
}

// Error is exposed by errors to get a Reason and Message.
type Error interface {
	error
	GetReason() Reason
	GetMessage() string
}

func GetReason(err error) Reason {
	if rerr := Error(nil); errors.As(err, &rerr) {
		return rerr.GetReason()
	}
	return ReasonUnknown
}

func GetMessage(err error) string {
	if rerr := Error(nil); errors.As(err, &rerr) {
		return rerr.GetMessage()
	}
	return ""
}

// NewAdmissionRefusedError creates a new Error whose Reason is AdmissionRefused.
func NewAdmissionRefusedError(message string) Error {
	return &internalError{
		Reason:  ReasonAdmissionRefused,
		Message: message,
	}
}

// IsAdmissionRefused returns true if the error is a Error whose Reason is AdmissionRefused.
func IsAdmissionRefused(err error) bool {
	return GetReason(err) == ReasonAdmissionRefused
}
