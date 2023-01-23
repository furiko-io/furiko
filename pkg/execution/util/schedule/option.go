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

package schedule

import (
	"k8s.io/utils/clock"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/config"
)

// Option is a kind of functional option.
type Option func(*Schedule)

// WithClock sets the clock to a custom one.
func WithClock(clock clock.Clock) Option {
	return func(s *Schedule) {
		s.clock = clock
	}
}

// Config knows how to return the cron config.
type Config interface {
	Cron() (*configv1alpha1.CronExecutionConfig, error)
}

// defaultConfig returns an empty config.
type defaultConfig struct{}

var _ Config = (*defaultConfig)(nil)

func (d *defaultConfig) Cron() (*configv1alpha1.CronExecutionConfig, error) {
	return config.DefaultCronExecutionConfig, nil
}

// WithConfigLoader sets the config loader to a custom one.
func WithConfigLoader(loader Config) Option {
	return func(s *Schedule) {
		s.cfg = loader
	}
}
