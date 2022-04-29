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
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/pkg/errors"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/streams"
)

// Prompt knows how to prompt the user for input, and returns the formatted
// option value.
type Prompt interface {
	Run() (interface{}, error)
}

// MakePrompt returns a general Prompt based on the given Option.
func MakePrompt(streams *streams.Streams, option execution.Option) (Prompt, error) {
	switch option.Type {
	case execution.OptionTypeBool:
		return NewBoolPrompt(streams, option), nil
	case execution.OptionTypeString:
		return NewStringPrompt(streams, option), nil
	case execution.OptionTypeSelect:
		return NewSelectPrompt(streams, option), nil
	case execution.OptionTypeMulti:
		return NewMultiPrompt(streams, option), nil
	case execution.OptionTypeDate:
		return NewDatePrompt(streams, option), nil
	}
	return nil, fmt.Errorf("unhandled option type: %v", option.Type)
}

type boolPrompt struct {
	s      *streams.Streams
	cfg    *execution.BoolOptionConfig
	survey *survey.Confirm
}

var _ Prompt = (*boolPrompt)(nil)

// NewBoolPrompt returns a new Prompt from a Bool option.
func NewBoolPrompt(s *streams.Streams, option execution.Option) Prompt {
	cfg := option.Bool
	if cfg == nil {
		cfg = &execution.BoolOptionConfig{}
	}
	return &boolPrompt{
		cfg: cfg,
		s:   s,
		survey: &survey.Confirm{
			Message: MakeLabel(option),
			Default: cfg.Default,
		},
	}
}

func (p *boolPrompt) Run() (interface{}, error) {
	var resp bool
	if err := survey.AskOne(
		p.survey,
		&resp,
		survey.WithStdio(p.s.In, p.s.Out, p.s.ErrOut),
	); err != nil {
		return nil, err
	}
	return resp, nil
}

type stringPrompt struct {
	s      *streams.Streams
	option execution.Option
	cfg    *execution.StringOptionConfig
	survey *survey.Input
}

var _ Prompt = (*stringPrompt)(nil)

// NewStringPrompt returns a Prompt from a String option.
func NewStringPrompt(s *streams.Streams, option execution.Option) Prompt {
	cfg := option.String
	if cfg == nil {
		cfg = &execution.StringOptionConfig{}
	}

	return &stringPrompt{
		s:      s,
		option: option,
		cfg:    cfg,
		survey: &survey.Input{
			Message: MakeLabel(option),
			Default: cfg.Default,
		},
	}
}

func (p *stringPrompt) Run() (interface{}, error) {
	var resp string
	if err := survey.AskOne(
		p.survey,
		&resp,
		survey.WithValidator(ValidateStringRequired(p.option.Required)),
		survey.WithStdio(p.s.In, p.s.Out, p.s.ErrOut),
	); err != nil {
		return nil, err
	}
	return resp, nil
}

type selectPrompt struct {
	s      *streams.Streams
	cfg    *execution.SelectOptionConfig
	survey *survey.Select
	option execution.Option
}

var _ Prompt = (*selectPrompt)(nil)

// NewSelectPrompt returns a new Prompt from a Select option.
func NewSelectPrompt(s *streams.Streams, option execution.Option) Prompt {
	cfg := option.Select
	if cfg == nil {
		cfg = &execution.SelectOptionConfig{}
	}
	return &selectPrompt{
		s:      s,
		cfg:    cfg,
		option: option,
		survey: &survey.Select{
			Message: MakeLabel(option),
			Options: cfg.Values,
			Default: cfg.Default,
		},
	}
}

func (p *selectPrompt) Run() (interface{}, error) {
	var resp string
	if err := survey.AskOne(
		p.survey,
		&resp,
		survey.WithValidator(ValidateStringRequired(p.option.Required)),
		survey.WithStdio(p.s.In, p.s.Out, p.s.ErrOut),
	); err != nil {
		return nil, err
	}
	return resp, nil
}

type multiPrompt struct {
	s      *streams.Streams
	cfg    *execution.MultiOptionConfig
	survey *survey.MultiSelect
	option execution.Option
}

var _ Prompt = (*multiPrompt)(nil)

// NewMultiPrompt returns a new Prompt from a Multi option.
func NewMultiPrompt(s *streams.Streams, option execution.Option) Prompt {
	cfg := option.Multi
	if cfg == nil {
		cfg = &execution.MultiOptionConfig{}
	}

	return &multiPrompt{
		s:      s,
		cfg:    cfg,
		option: option,
		survey: &survey.MultiSelect{
			Message: MakeLabel(option),
			Options: cfg.Values,
			Default: cfg.Default,
		},
	}
}

func (p *multiPrompt) Run() (interface{}, error) {
	var value []string
	if err := survey.AskOne(
		p.survey,
		&value,
		survey.WithValidator(ValidateMultiRequired(p.option.Required)),
		survey.WithStdio(p.s.In, p.s.Out, p.s.ErrOut),
	); err != nil {
		return nil, err
	}
	return value, nil
}

type datePrompt struct {
	s      *streams.Streams
	option execution.Option
	cfg    *execution.DateOptionConfig
	survey *survey.Input
}

var _ Prompt = (*datePrompt)(nil)

// NewDatePrompt returns a new Prompt from a Date option.
func NewDatePrompt(s *streams.Streams, option execution.Option) Prompt {
	cfg := option.Date
	if cfg == nil {
		cfg = &execution.DateOptionConfig{}
	}

	return &datePrompt{
		s:      s,
		cfg:    cfg,
		option: option,
		survey: &survey.Input{
			Message: MakeLabel(option),
			Help:    "Enter a RFC3339-formatted time string.",
			Suggest: func(toComplete string) []string {
				return []string{
					time.Now().Format(time.RFC3339),
				}
			},
		},
	}
}

func (p *datePrompt) Run() (interface{}, error) {
	var value string
	if err := survey.AskOne(
		p.survey,
		&value,
		survey.WithValidator(ValidateStringRequired(p.option.Required)),
		survey.WithValidator(ValidateDate),
		survey.WithStdio(p.s.In, p.s.Out, p.s.ErrOut),
	); err != nil {
		return nil, err
	}
	return value, nil
}

// IsInterruptError returns true if the given error is a keyboard interrupt
// error raised by a Prompt.
func IsInterruptError(err error) bool {
	// We only use github.com/AlecAivazis/survey, so we can just check for the error
	// that is raised by that package.
	return errors.Is(err, terminal.InterruptErr)
}
