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

package printer

import (
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/furiko-io/furiko/pkg/cli/formatter"
)

// TablePrinter prints tables with header and body rows.
// TODO(irvinlim): Maybe we could use printers.TablePrinter instead of rolling our own.
type TablePrinter struct {
	out io.Writer
}

func NewTablePrinter(out io.Writer) *TablePrinter {
	return &TablePrinter{out: out}
}

// Print the header and each row to standard output.
func (p *TablePrinter) Print(header []string, rows [][]string) {
	t := table.NewWriter()
	t.SetOutputMirror(p.out)
	style := table.StyleLight
	style.Options = table.Options{
		SeparateColumns: true,
	}
	style.Box.PaddingLeft = ""
	style.Box.PaddingRight = ""
	style.Box.MiddleVertical = "   "
	t.SetStyle(style)
	t.AppendHeader(p.makeTableRow(header))
	t.AppendRows(p.makeTableRows(rows))
	t.Render()
}

func (p *TablePrinter) makeTableRows(rows [][]string) []table.Row {
	output := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		output = append(output, p.makeTableRow(row))
	}
	return output
}

func (p *TablePrinter) makeTableRow(cells []string) table.Row {
	output := make(table.Row, 0, len(cells))
	for _, cell := range cells {
		output = append(output, cell)
	}
	return output
}

// MaybeAppendTimeAgo is a helper that appends a key-value description pair to
// kvs if t is non-zero, formatted as a time with relative duration from now.
func MaybeAppendTimeAgo(kvs [][]string, key string, t *metav1.Time) [][]string {
	if !t.IsZero() {
		kvs = append(kvs, []string{key, formatter.FormatTimeWithTimeAgo(t)})
	}
	return kvs
}
