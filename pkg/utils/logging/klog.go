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

package logging

import (
	"flag"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

// SetLogLevel updates the klog log level. Any subsequent klog calls will then
// use the new log level.
func SetLogLevel(level klog.Level) error {
	flagSet := flag.NewFlagSet("", flag.ContinueOnError)
	klog.InitFlags(flagSet)
	if err := flagSet.Parse([]string{fmt.Sprintf("-v=%v", level)}); err != nil {
		return errors.Wrapf(err, "cannot parse log level")
	}
	return nil
}
