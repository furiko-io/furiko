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

package validation

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var (
	fldPath = field.NewPath("spec")
)

func TestToInternalErrorList(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want field.ErrorList
	}{
		{
			name: "nil error",
		},
		{
			name: "non-nil error",
			err:  assert.AnError,
			want: field.ErrorList{
				field.InternalError(fldPath, assert.AnError),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := ToInternalErrorList(fldPath, tt.err)
			if !cmp.Equal(got, tt.want, cmpopts.EquateEmpty()) {
				t.Errorf("ToInternalErrorList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateImmutableField(t *testing.T) {
	tests := []struct {
		name   string
		newVal interface{}
		oldVal interface{}
		msg    string
		want   field.ErrorList
	}{
		{
			name: "nil values",
		},
		{
			name:   "different integer values",
			newVal: 1,
			oldVal: 0,
			msg:    "custom error message",
			want: field.ErrorList{
				field.Invalid(fldPath, 1, "custom error message"),
			},
		},
		{
			name:   "equal integer values",
			newVal: 1,
			oldVal: 1,
			msg:    "custom error message",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateImmutableField(tt.newVal, tt.oldVal, fldPath, tt.msg)
			if !cmp.Equal(got, tt.want, cmpopts.EquateEmpty()) {
				t.Errorf("ValidateImmutableField(%v, %v, %v, %v) = %v, want %v",
					tt.newVal, tt.oldVal, fldPath, tt.msg, got, tt.want)
			}
		})
	}
}

func TestValidateGTE(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		minimum int64
		want    field.ErrorList
	}{
		{
			name:    "greater than",
			value:   11,
			minimum: 10,
		},
		{
			name:    "equal to",
			value:   10,
			minimum: 10,
		},
		{
			name:    "less than",
			value:   9,
			minimum: 10,
			want: field.ErrorList{
				field.Invalid(fldPath, int64(9), "must be greater than or equal to 10"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateGTE(tt.value, tt.minimum, fldPath); !cmp.Equal(got, tt.want, cmpopts.EquateEmpty()) {
				t.Errorf("ValidateGTE(%v, %v, %v) = %v, want %v", tt.value, tt.minimum, fldPath, got, tt.want)
			}
		})
	}
}

func TestValidateGT(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		minimum int64
		want    field.ErrorList
	}{
		{
			name:    "greater than",
			value:   11,
			minimum: 10,
		},
		{
			name:    "equal to",
			value:   10,
			minimum: 10,
			want: field.ErrorList{
				field.Invalid(fldPath, int64(10), "must be greater than 10"),
			},
		},
		{
			name:    "less than",
			value:   9,
			minimum: 10,
			want: field.ErrorList{
				field.Invalid(fldPath, int64(9), "must be greater than 10"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateGT(tt.value, tt.minimum, fldPath); !cmp.Equal(got, tt.want, cmpopts.EquateEmpty()) {
				t.Errorf("ValidateGT(%v, %v, %v) = %v, want %v", tt.value, tt.minimum, fldPath, got, tt.want)
			}
		})
	}
}

func TestValidateLTE(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		minimum int64
		want    field.ErrorList
	}{
		{
			name:    "greater than",
			value:   11,
			minimum: 10,
			want: field.ErrorList{
				field.Invalid(fldPath, int64(11), "must be less than or equal to 10"),
			},
		},
		{
			name:    "equal to",
			value:   10,
			minimum: 10,
		},
		{
			name:    "less than",
			value:   9,
			minimum: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateLTE(tt.value, tt.minimum, fldPath); !cmp.Equal(got, tt.want, cmpopts.EquateEmpty()) {
				t.Errorf("ValidateLTE(%v, %v, %v) = %v, want %v", tt.value, tt.minimum, fldPath, got, tt.want)
			}
		})
	}
}

func TestValidateLT(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		minimum int64
		want    field.ErrorList
	}{
		{
			name:    "greater than",
			value:   11,
			minimum: 10,
			want: field.ErrorList{
				field.Invalid(fldPath, int64(11), "must be less than 10"),
			},
		},
		{
			name:    "equal to",
			value:   10,
			minimum: 10,
			want: field.ErrorList{
				field.Invalid(fldPath, int64(10), "must be less than 10"),
			},
		},
		{
			name:    "less than",
			value:   9,
			minimum: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateLT(tt.value, tt.minimum, fldPath); !cmp.Equal(got, tt.want, cmpopts.EquateEmpty()) {
				t.Errorf("ValidateLT(%v, %v, %v) = %v, want %v", tt.value, tt.minimum, fldPath, got, tt.want)
			}
		})
	}
}
