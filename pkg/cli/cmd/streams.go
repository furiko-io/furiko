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

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Streams encapsulates I/O streams.
type Streams struct {
	genericclioptions.IOStreams
}

// NewStdStreams returns a Streams object bound to stdin, stdout, stderr.
func NewStdStreams() *Streams {
	return NewStreams(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
}

// NewStreams returns a Streams object from an underlying genericclioptions.IOStreams.
func NewStreams(streams genericclioptions.IOStreams) *Streams {
	return &Streams{
		IOStreams: streams,
	}
}

// SetCmdOutput updates the output streams of the command to itself.
func (s *Streams) SetCmdOutput(cmd *cobra.Command) {
	cmd.SetOut(s.Out)
	cmd.SetErr(s.ErrOut)
}

// Printf writes to the output stream.
func (s *Streams) Printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(s.Out, format, args...)
}

// Println writes to the output stream.
func (s *Streams) Println(args ...interface{}) {
	_, _ = fmt.Fprintln(s.Out, args...)
}

// Print writes to the output stream.
func (s *Streams) Print(args ...interface{}) {
	_, _ = fmt.Fprint(s.Out, args...)
}
