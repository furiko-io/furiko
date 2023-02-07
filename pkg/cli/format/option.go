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

package format

// Option represents functional options for various format functions.
type Option func(c *Config)

// Config represents formatting configuration values.
type Config struct {
	// DefaultValue is used in case of a zero value.
	DefaultValue string
}

// WithDefaultValue specifies a default value for the formatter.
func WithDefaultValue(value string) Option {
	return func(c *Config) {
		c.DefaultValue = value
	}
}

func parseConfigFromOptions(opts []Option) *Config {
	c := &Config{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
