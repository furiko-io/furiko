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

package streams

import (
	"fmt"
	"io"
	"os"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Streams encapsulates I/O streams.
type Streams struct {
	In     terminal.FileReader
	Out    terminal.FileWriter
	ErrOut io.Writer
}

// NewStdStreams returns a Streams object bound to stdin, stdout, stderr.
func NewStdStreams() *Streams {
	return &Streams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

// NewStreams returns a Streams object from an underlying genericclioptions.IOStreams.
func NewStreams(streams genericclioptions.IOStreams) *Streams {
	return &Streams{
		In:     NewFileReader(streams.In, 0),
		Out:    NewFileWriter(streams.Out, 1),
		ErrOut: streams.ErrOut,
	}
}

// NewTTYStreams returns a Streams object from a TTY file.
func NewTTYStreams(tty *os.File) *Streams {
	return &Streams{
		In:     tty,
		Out:    tty,
		ErrOut: tty,
	}
}

// SetCmdOutput updates the output streams of the command to itself.
func (s *Streams) SetCmdOutput(cmd *cobra.Command) {
	// Set cobra output
	cmd.SetOut(s.Out)
	cmd.SetErr(s.ErrOut)

	// Set color output
	color.Output = s.Out
	color.Error = s.ErrOut
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
