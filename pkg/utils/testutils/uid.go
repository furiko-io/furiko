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

package testutils

import (
	"hash/fnv"
	"math/rand"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
)

// MakeUID returns a deterministically generated UID given an input string.
// nolint:gosec
func MakeUID(name string) types.UID {
	h := fnv.New64a()
	if _, err := h.Write([]byte(name)); err != nil {
		panic(err)
	}
	rnd := rand.New(rand.NewSource(int64(h.Sum64())))
	uid, err := uuid.NewRandomFromReader(rnd)
	if err != nil {
		panic(err)
	}
	return types.UID(uid.String())
}
