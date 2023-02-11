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
	"fmt"
	"os"
	"strings"

	"github.com/kr/text"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/printer"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

const (
	DefaultErrorExitCode = 1
)

var (
	ctrlContext controllercontext.Context
)

type RunEFunc func(cmd *cobra.Command, args []string) error

// RunAllE composes multiple RunEFunc together.
func RunAllE(funcs ...RunEFunc) RunEFunc {
	return func(cmd *cobra.Command, args []string) error {
		for _, f := range funcs {
			if err := f(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}

// NewContext returns a common context from the cobra command.
func NewContext(cmd *cobra.Command) (controllercontext.Context, error) {
	kubeconfig, err := GetKubeConfig(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get kubeconfig")
	}
	return controllercontext.NewForConfig(kubeconfig, &configv1alpha1.BootstrapConfigSpec{})
}

// SetupCtrlContext sets up the common context.
// TODO(irvinlim): We currently reuse controllercontext, but most of it is unusable for CLI interfaces.
// We should create a new common context as needed.
func SetupCtrlContext(cmd *cobra.Command) error {
	// Already set up previously.
	if ctrlContext != nil {
		return nil
	}

	newContext, err := NewContext(cmd)
	if err != nil {
		return err
	}
	ctrlContext = newContext
	return nil
}

// GetCtrlContext retrieves the common context.
func GetCtrlContext() controllercontext.Context {
	return ctrlContext
}

// SetCtrlContext explicitly sets the common context.
func SetCtrlContext(cc controllercontext.Context) {
	ctrlContext = cc
}

// GetNamespace returns the namespace to use depending on what was defined in the flags.
func GetNamespace(cmd *cobra.Command) (string, error) {
	// If --all-namespaces is defined on the command, use it first.
	if allNamespaces, ok := GetFlagBoolIfExists(cmd, "all-namespaces"); ok && allNamespaces {
		return metav1.NamespaceAll, nil
	}

	// Read the --namespace flag if specified.
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return "", errors.Wrapf(err, "cannot get value of --namespace")
	}

	// Otherwise, fall back to the default namespace.
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	return namespace, nil
}

// GetOutputFormat returns the output format as parsed by the flag.
func GetOutputFormat(cmd *cobra.Command) printer.OutputFormat {
	v := GetFlagString(cmd, "output")
	output := printer.OutputFormat(v)
	klog.V(4).InfoS("using output format", "format", output)
	return output
}

// PrepareExample replaces the root command name and indents all lines.
func PrepareExample(example string) string {
	example = strings.TrimPrefix(example, "\n")
	example = strings.ReplaceAll(example, "{{.CommandName}}", CommandName)
	return text.Indent(example, "  ")
}

// Fatal terminates the command immediately and writes to stderr.
func Fatal(err error, code int) {
	msg := err.Error()
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		_, _ = fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(code)
}
