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

package main

import (
	"flag"
	"time"
)

const (
	defaultStartupTimeout  = 3 * time.Minute
	defaultTeardownTimeout = 2 * time.Minute
)

var (
	configFile      string
	startupTimeout  time.Duration
	teardownTimeout time.Duration
)

func init() {
	flag.StringVar(&configFile, "config", "",
		"The controller will load its bootstrap configuration from this file.")
	flag.DurationVar(&startupTimeout, "startup-timeout", defaultStartupTimeout,
		"Timeout to start up all controllers, set to negative for no timeout")
	flag.DurationVar(&teardownTimeout, "teardown-timeout", defaultTeardownTimeout,
		"Timeout to tear down all controllers, before it will forcibly quit")
}
