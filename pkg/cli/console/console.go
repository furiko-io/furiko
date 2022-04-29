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

package console

import (
	"io"

	"github.com/Netflix/go-expect"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/pkg/errors"
)

type Console struct {
	*expect.Console
}

// NewConsole returns a Console for testing PTY handling.
// All PTY output will be piped to w.
func NewConsole(w io.Writer) (*Console, error) {
	ptty, tty, err := pty.Open()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open pty")
	}

	term := vt10x.New(vt10x.WithWriter(tty))
	console, err := expect.NewConsole(
		expect.WithStdin(ptty),
		expect.WithStdout(term, w),
		expect.WithCloser(ptty, tty),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create console")
	}

	c := &Console{
		Console: console,
	}

	return c, nil
}

func (c *Console) Run(f func(c *Console)) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if f != nil {
			f(c)
		}

		// Wait until the stdin is closed.
		_, _ = c.ExpectEOF()
	}()
	return done
}

func (c *Console) ExpectString(s string) {
	_, _ = c.Console.ExpectString(s)
}

func (c *Console) SendLine(s string) {
	_, _ = c.Console.SendLine(s)
}
