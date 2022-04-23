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
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Section contains a section name and a list of description key-value pairs.
type Section struct {
	Name  string
	Descs [][]string
}

type SectionGenerator func() (Section, error)

// ToSectionGenerator converts a Section into a SectionGenerator.
func ToSectionGenerator(section Section) SectionGenerator {
	return func() (Section, error) {
		return section, nil
	}
}

// DescriptionPrinter prints key-value pairs, separated by sections.
type DescriptionPrinter struct {
	out        io.Writer
	minSpacing int
}

func NewDescriptionPrinter(out io.Writer) *DescriptionPrinter {
	return &DescriptionPrinter{
		out:        out,
		minSpacing: 2,
	}
}

func NewStdoutDescriptionPrinter() *DescriptionPrinter {
	return NewDescriptionPrinter(os.Stdout)
}

func (p *DescriptionPrinter) Header(header string) {
	p.Print(p.getHeader(header))
}

func (p *DescriptionPrinter) Descriptions(kvs [][]string) {
	p.Print(kvs)
}

func (p *DescriptionPrinter) NewLine() {
	p.Print([][]string{{}})
}

func (p *DescriptionPrinter) Divider() {
	p.Print([][]string{{"---"}})
}

func (p *DescriptionPrinter) getHeader(header string) [][]string {
	return [][]string{
		{color.CyanString(strings.ToUpper(header))},
	}
}

// Print prints each of the given key-value pairs.
func (p *DescriptionPrinter) Print(kvs [][]string) {
	var maxLen int
	for _, kv := range kvs {
		if len(kv) == 0 {
			continue
		}
		if len(kv[0]) > maxLen {
			maxLen = len(kv[0])
		}
	}

	// Add a minimum space between keys and values, plus one for the colon.
	maxLen += p.minSpacing + 1

	for _, kv := range kvs {
		// Add empty line.
		if len(kv) == 0 {
			_, _ = fmt.Fprintln(p.out, "")
			continue
		}

		// Add single key without colon.
		key := kv[0]
		if len(kv) == 1 {
			_, _ = fmt.Fprintln(p.out, key)
			continue
		}

		// Print key-value pair.
		value := kv[1]
		_, _ = fmt.Fprintln(p.out, text.Pad(key+":", maxLen, ' ')+value)
	}
}

// TablePrinter prints tables with header and body rows.
type TablePrinter struct {
	out io.Writer
}

func NewTablePrinter(out io.Writer) *TablePrinter {
	return &TablePrinter{out: out}
}

func NewStdoutTablePrinter() *TablePrinter {
	return NewTablePrinter(os.Stdout)
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
		kvs = append(kvs, []string{key, FormatTimeWithTimeAgo(t)})
	}
	return kvs
}
