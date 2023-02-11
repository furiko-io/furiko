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

package common

import (
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

// GetFlagBool gets the boolean value of a flag.
func GetFlagBool(cmd *cobra.Command, flag string) bool {
	b, err := cmd.Flags().GetBool(flag)
	if err != nil {
		klog.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}
	return b
}

// GetFlagBoolIfExists gets the boolean value of a flag if it exists.
func GetFlagBoolIfExists(cmd *cobra.Command, flag string) (val bool, ok bool) {
	if cmd.Flags().Lookup(flag) != nil {
		val := GetFlagBool(cmd, flag)
		return val, true
	}
	return false, false
}

// GetFlagString gets the string value of a flag.
func GetFlagString(cmd *cobra.Command, flag string) string {
	v, err := cmd.Flags().GetString(flag)
	if err != nil {
		klog.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}
	return v
}

// GetFlagInt64 gets the int64 value of a flag.
func GetFlagInt64(cmd *cobra.Command, flag string) int64 {
	v, err := cmd.Flags().GetInt64(flag)
	if err != nil {
		klog.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}
	return v
}
