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

package sets

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

// Re-export types and functions from k8s.io/apimachinery/pkg/util/sets to
// maximize drop-in compatibility.

type (
	String = sets.String
	Int    = sets.Int
	Int32  = sets.Int32
	Int64  = sets.Int64
	Byte   = sets.Byte
	Empty  = sets.Empty
)

var (
	NewString = sets.NewString
	NewInt    = sets.NewInt
	NewInt32  = sets.NewInt32
	NewInt64  = sets.NewInt64
	NewByte   = sets.NewByte
)
