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

package leaderelection

import (
	"time"

	timeutil "github.com/furiko-io/furiko/pkg/utils/time"
)

type Config struct {
	LeaseName      string
	LeaseNamespace string
	LeaseDuration  time.Duration
	RenewDeadline  time.Duration
	RetryPeriod    time.Duration
}

var (
	DefaultConfig = &Config{
		LeaseNamespace: "furiko-system",
		LeaseDuration:  30 * time.Second,
		RenewDeadline:  15 * time.Second,
		RetryPeriod:    5 * time.Second,
	}
)

func (c *Config) PrepareValues() *Config {
	cfg := c
	if cfg == nil {
		cfg = &Config{}
	}
	return &Config{
		LeaseDuration:  durationDefaulting(cfg.LeaseDuration, DefaultConfig.LeaseDuration),
		RenewDeadline:  durationDefaulting(cfg.RenewDeadline, DefaultConfig.RenewDeadline),
		RetryPeriod:    durationDefaulting(cfg.RetryPeriod, DefaultConfig.RetryPeriod),
		LeaseName:      cfg.LeaseName, // no defaults provided
		LeaseNamespace: stringDefaulting(cfg.LeaseNamespace, DefaultConfig.LeaseNamespace),
	}
}

func stringDefaulting(value, defaultValue string) string {
	if value == "" {
		value = defaultValue
	}
	return value
}

func durationDefaulting(value, defaultValue time.Duration) time.Duration {
	value = timeutil.DurationMax(0, value)
	if value == 0 {
		value = defaultValue
	}
	return value
}
