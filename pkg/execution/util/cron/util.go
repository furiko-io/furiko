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
	"github.com/pkg/errors"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
)

// NewExpressionFromCronSchedule returns a new Expression after parsing all expressions from *execution.CronSchedule.
func NewExpressionFromCronSchedule(spec *execution.CronSchedule, parser *Parser, hashID string) (Expression, error) {
	if spec == nil {
		spec = &execution.CronSchedule{}
	}

	cronLines := spec.GetExpressions()
	expressions := make([]Expression, 0, len(cronLines))
	for _, cronLine := range cronLines {
		expr, err := parser.Parse(cronLine, hashID)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot parse cron expression: %v", cronLine)
		}
		expressions = append(expressions, expr)
	}

	return NewMultiExpression(expressions...), nil
}
