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

package cron

import (
	"time"

	timeutil "github.com/furiko-io/furiko/pkg/utils/time"
)

// Expression is implemented by cron expressions.
type Expression interface {
	// Next returns the closest time instant immediately following `fromTime` which
	// matches the cron expression `expr`.
	Next(fromTime time.Time) time.Time
}

type multiExpression struct {
	expressions []Expression
}

var _ Expression = (*multiExpression)(nil)

// NewMultiExpression wraps multiple Expression into a single one.
func NewMultiExpression(expressions ...Expression) Expression {
	return &multiExpression{
		expressions: expressions,
	}
}

// Next returns the next earliest timestamp for all expressions.
// If none of the expressions have a next timestamp, a zero time is returned.
// If there are no expressions, a zero time is also returned.
func (m *multiExpression) Next(fromTime time.Time) time.Time {
	var earliestNext time.Time
	for _, expression := range m.expressions {
		next := expression.Next(fromTime)
		if !next.IsZero() {
			earliestNext = timeutil.MinNonZero(next, earliestNext)
		}
	}
	return earliestNext
}
