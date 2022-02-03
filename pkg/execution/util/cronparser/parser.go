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

package cronparser

import (
	"github.com/furiko-io/cronexpr"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
)

// Parser wraps the raw cronexpr parser to encapsulate common configuration.
type Parser struct {
	format    cronexpr.CronFormat
	hashNames bool
	opts      []cronexpr.ParseOption
}

func NewParser(cfg *configv1.CronControllerConfig) *Parser {
	format := cronexpr.CronFormatStandard
	if cronexpr.CronFormat(cfg.CronFormat) == cronexpr.CronFormatQuartz {
		format = cronexpr.CronFormatQuartz
	}

	hashNames := unwrapBool(cfg.CronHashNames, true)

	var parseOpts []cronexpr.ParseOption
	if hashNames {
		if unwrapBool(cfg.CronHashSecondsByDefault, false) {
			parseOpts = append(parseOpts, cronexpr.WithHashEmptySeconds())
		}
		if unwrapBool(cfg.CronHashFields, true) {
			parseOpts = append(parseOpts, cronexpr.WithHashFields())
		}
	}

	return &Parser{
		format:    format,
		hashNames: hashNames,
		opts:      parseOpts,
	}
}

func (p *Parser) Parse(cronLine string, hashID string) (*cronexpr.Expression, error) {
	opts := p.opts
	if p.hashNames {
		newOpts := make([]cronexpr.ParseOption, 0, len(p.opts)+1)
		newOpts = append(newOpts, cronexpr.WithHash(hashID))
		newOpts = append(newOpts, p.opts...)
		opts = newOpts
	}

	return cronexpr.ParseForFormat(p.format, cronLine, opts...)
}

func unwrapBool(b *bool, defaultBool bool) bool {
	val := defaultBool
	if b != nil {
		val = *b
	}
	return val
}
