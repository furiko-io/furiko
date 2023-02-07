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
	"github.com/furiko-io/cronexpr"
	"k8s.io/utils/pointer"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
)

// Parser wraps the raw cronexpr parser to encapsulate common configuration.
type Parser struct {
	// Specifies the CronFormat to use.
	Format cronexpr.CronFormat

	// If true, names will be hashed when passed to Parse.
	HashNames bool

	// Optionally specify the raw ParseOption to pass to the cronexpr parser
	ParseOptions []cronexpr.ParseOption
}

func NewParserFromConfig(cfg *configv1alpha1.CronExecutionConfig) *Parser {
	format := cronexpr.CronFormatStandard
	if cronexpr.CronFormat(cfg.CronFormat) == cronexpr.CronFormatQuartz {
		format = cronexpr.CronFormatQuartz
	}

	hashNames := pointer.BoolDeref(cfg.CronHashNames, true)

	var parseOpts []cronexpr.ParseOption
	if hashNames {
		if pointer.BoolDeref(cfg.CronHashSecondsByDefault, false) {
			parseOpts = append(parseOpts, cronexpr.WithHashEmptySeconds())
		}
		if pointer.BoolDeref(cfg.CronHashFields, true) {
			parseOpts = append(parseOpts, cronexpr.WithHashFields())
		}
	}

	return &Parser{
		Format:       format,
		HashNames:    hashNames,
		ParseOptions: parseOpts,
	}
}

// Parse the cronLine, with an optional hashID if HashNames is enabled.
func (p *Parser) Parse(cronLine string, hashID string) (*cronexpr.Expression, error) {
	opts := p.ParseOptions
	if p.HashNames {
		newOpts := make([]cronexpr.ParseOption, 0, len(p.ParseOptions)+1)
		newOpts = append(newOpts, cronexpr.WithHash(hashID))
		newOpts = append(newOpts, p.ParseOptions...)
		opts = newOpts
	}

	return cronexpr.ParseForFormat(p.Format, cronLine, opts...)
}
