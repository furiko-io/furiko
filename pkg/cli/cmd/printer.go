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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeprinters "k8s.io/cli-runtime/pkg/printers"
	"k8s.io/kubernetes/pkg/printers"
)

// Printer knows how to print multiple items.
type Printer interface {
	PrintObj(obj runtime.Object) error
	PrintItems(items []runtime.Object) error
}

// GetPrinter returns the Printer associated with output.
func GetPrinter(output OutputFormat, out io.Writer) (Printer, error) {
	switch output {
	case OutputFormatYAML:
		return NewYAMLPrinter(out), nil
	case OutputFormatJSON:
		return NewJSONPrinter(out), nil
	case OutputFormatName:
		return NewNamePrinter(out), nil
	}
	return nil, fmt.Errorf("invalid format %v specified, must be one of: %v",
		output, strings.Join(GetAllOutputFormatStrings(), ","))
}

// PrintObject will print a single object using the printer corresponding to the
// specified output format.
func PrintObject(output OutputFormat, out io.Writer, object runtime.Object) error {
	p, err := GetPrinter(output, out)
	if err != nil {
		return err
	}
	return p.PrintObj(object)
}

// PrintItems will print all items using the printer corresponding to the specified
// output format.
func PrintItems(output OutputFormat, out io.Writer, items []runtime.Object) error {
	p, err := GetPrinter(output, out)
	if err != nil {
		return err
	}
	return p.PrintItems(items)
}

// NamePrinter prints names.
type NamePrinter struct {
	out io.Writer
}

// NewNamePrinter returns a new NamePrinter that prints names to the given
// output.
func NewNamePrinter(out io.Writer) *NamePrinter {
	return &NamePrinter{out: out}
}

func (p *NamePrinter) PrintItems(items []runtime.Object) error {
	for _, item := range items {
		if err := p.PrintObj(item); err != nil {
			return err
		}
	}
	return nil
}

func (p *NamePrinter) PrintObj(item runtime.Object) error {
	obj, ok := item.(metav1.Object)
	if !ok {
		return fmt.Errorf("cannot convert to metav1.Object: %T", item)
	}
	_, err := fmt.Fprintln(p.out, obj.GetName())
	return err
}

// JSONPrinter prints JSON.
type JSONPrinter struct {
	*MultiPrinter
}

// NewJSONPrinter returns a new JSONPrinter that prints items to the given
// output.
func NewJSONPrinter(out io.Writer) *JSONPrinter {
	return &JSONPrinter{
		MultiPrinter: NewMultiPrinter(&runtimeprinters.JSONPrinter{}, out),
	}
}

// YAMLPrinter prints YAML.
type YAMLPrinter struct {
	*MultiPrinter
}

// NewYAMLPrinter returns a new YAMLPrinter that prints items to the given
// output.
func NewYAMLPrinter(out io.Writer) *YAMLPrinter {
	return &YAMLPrinter{
		MultiPrinter: NewMultiPrinter(&runtimeprinters.YAMLPrinter{}, out),
	}
}

// MultiPrinter is a generic printer that handles single and multiple items
// specially.
type MultiPrinter struct {
	objPrinter printers.ResourcePrinter
	out        io.Writer
}

func NewMultiPrinter(objPrinter printers.ResourcePrinter, out io.Writer) *MultiPrinter {
	// Skip printing managed fields by default.
	objPrinter = &runtimeprinters.OmitManagedFieldsPrinter{
		Delegate: objPrinter,
	}

	return &MultiPrinter{
		objPrinter: objPrinter,
		out:        out,
	}
}

func (p *MultiPrinter) PrintObj(obj runtime.Object) error {
	return p.objPrinter.PrintObj(obj, p.out)
}

func (p *MultiPrinter) PrintItems(items []runtime.Object) error {
	// If there is only a single item, print that item only.
	if len(items) == 1 {
		return p.objPrinter.PrintObj(items[0], p.out)
	}

	// Use a List to print multiple objects instead.
	extensions := make([]runtime.RawExtension, 0, len(items))
	for _, item := range items {
		extensions = append(extensions, runtime.RawExtension{
			Object: item,
		})
	}

	list := &metav1.List{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.SchemeGroupVersion.Version,
		},
		Items: extensions,
	}

	return p.objPrinter.PrintObj(list, p.out)
}
