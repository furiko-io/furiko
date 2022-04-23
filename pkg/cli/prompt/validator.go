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

package prompt

import (
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
)

// ValidateDate is a survey.Validator that validates if a date is valid.
func ValidateDate(ans interface{}) error {
	v, ok := ans.(string)
	if !ok {
		return errors.New("not a string")
	}
	if v != "" {
		if _, err := time.Parse(time.RFC3339, v); err != nil {
			return err
		}
	}
	return nil
}

// ValidateStringRequired returns a survey.Validator that validates a string
// answer if required.
func ValidateStringRequired(required bool) survey.Validator {
	return func(ans interface{}) error {
		if !required {
			return nil
		}
		v, ok := ans.(string)
		if !ok {
			return errors.New("not a string")
		}
		if v == "" {
			return errors.New("value is required")
		}
		return nil
	}
}

// ValidateMultiRequired returns a survey.Validator that validates a multiselect
// answer if required.
func ValidateMultiRequired(required bool) survey.Validator {
	return func(ans interface{}) error {
		if !required {
			return nil
		}
		v, ok := ans.([]string)
		if !ok {
			return errors.New("not a string slice")
		}
		if len(v) == 0 {
			return errors.New("at least one value must be selected")
		}
		return nil
	}
}
