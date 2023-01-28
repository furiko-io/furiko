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
	"fmt"
	"io"
	"strings"

	"github.com/liggitt/tabwriter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/furiko-io/furiko/pkg/cli/formatter"
)

// TablePrinter prints tables with header and body rows.
type TablePrinter struct {
	writer         *tabwriter.Writer
	headersPrinted bool
}

func NewTablePrinter(out io.Writer) *TablePrinter {
	return &TablePrinter{
		writer: printers.GetNewTabWriter(out),
	}
}

// Print the header and each row to standard output.
// By default, headers are printed at most once.
func (p *TablePrinter) Print(header []string, rows [][]string) {
	if len(header) > 0 && !p.headersPrinted {
		p.printRow(header)
		p.headersPrinted = true
	}
	for _, row := range rows {
		p.printRow(row)
	}
	p.writer.Flush()
}

func (p *TablePrinter) printRow(cols []string) {
	fmt.Fprint(p.writer, strings.Join(cols, "\t"))
	_, _ = p.writer.Write([]byte("\n"))
}

// MaybeAppendTimeAgo is a helper that appends a key-value description pair to
// kvs if t is non-zero, formatted as a time with relative duration from now.
func MaybeAppendTimeAgo(kvs [][]string, key string, t *metav1.Time) [][]string {
	if !t.IsZero() {
		kvs = append(kvs, []string{key, formatter.FormatTimeWithTimeAgo(t)})
	}
	return kvs
}
