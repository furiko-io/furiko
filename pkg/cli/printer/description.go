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

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/text"
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
