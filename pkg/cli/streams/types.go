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
	"io"

	"github.com/AlecAivazis/survey/v2/terminal"
)

type FileReader struct {
	io.Reader
	fd uintptr
}

var _ terminal.FileReader = (*FileReader)(nil)

func NewFileReader(reader io.Reader, fd uint) *FileReader {
	return &FileReader{Reader: reader, fd: uintptr(fd)}
}

func (f *FileReader) Fd() uintptr {
	return f.fd
}

type FileWriter struct {
	io.Writer
	fd uintptr
}

var _ terminal.FileWriter = (*FileWriter)(nil)

func NewFileWriter(reader io.Writer, fd uint) *FileWriter {
	return &FileWriter{Writer: reader, fd: uintptr(fd)}
}

func (f *FileWriter) Fd() uintptr {
	return f.fd
}
