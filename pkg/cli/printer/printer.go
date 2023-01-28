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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimeprinters "k8s.io/cli-runtime/pkg/printers"
	"k8s.io/kubernetes/pkg/printers"
)

// Object is a runtime object that also has a Kind.
type Object interface {
	schema.ObjectKind
	runtime.Object
}

// Initialize printers once at startup, otherwise multiple GetPrinter calls will
// lose internal state.
var (
	// Don't show managed fields by default for YAML.
	yamlPrinter = &runtimeprinters.OmitManagedFieldsPrinter{
		Delegate: &runtimeprinters.YAMLPrinter{},
	}

	// Don't show managed fields by default for JSON.
	jsonPrinter = &runtimeprinters.OmitManagedFieldsPrinter{
		Delegate: &runtimeprinters.JSONPrinter{},
	}

	namePrinter = NewNamePrinter()
)

// GetPrinter returns the ResourcePrinter associated with the given output format. Most
// implementations are taken from cli-runtime itself, so we don't need to really
// implement too much.
func GetPrinter(output OutputFormat) (printers.ResourcePrinter, error) {
	var printer printers.ResourcePrinter

	switch output {
	case OutputFormatYAML:
		printer = yamlPrinter
	case OutputFormatJSON:
		printer = jsonPrinter
	case OutputFormatName:
		printer = namePrinter
	default:
		return nil, fmt.Errorf("invalid format %v specified, must be one of: %v",
			output, strings.Join(GetAllOutputFormatStrings(), ","))
	}

	return printer, nil
}

// PrintObject will print a single object using the printer corresponding to the
// specified output format.
func PrintObject(gvk schema.GroupVersionKind, output OutputFormat, out io.Writer, object Object) error {
	p, err := GetPrinter(output)
	if err != nil {
		return err
	}

	// Set the GVK of the object before passing to the printer.
	// TODO(irvinlim): This is duplicated with PrintObjects, but is needed
	//  in case we need to support printing single object.
	object.SetGroupVersionKind(gvk)

	return p.PrintObj(object, out)
}

// PrintObjects will print all objects individually.
func PrintObjects(gvk schema.GroupVersionKind, output OutputFormat, out io.Writer, objs []Object) error {
	for _, obj := range objs {
		if err := PrintObject(gvk, output, out, obj); err != nil {
			return err
		}
	}
	return nil
}
