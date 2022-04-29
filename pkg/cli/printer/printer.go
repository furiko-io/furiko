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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// GetPrinter returns the ResourcePrinter associated with output. Most
// implementations are taken from cli-runtime itself, so we don't need to really
// implement too much.
func GetPrinter(output OutputFormat) (printers.ResourcePrinter, error) {
	var printer printers.ResourcePrinter

	switch output {
	case OutputFormatYAML:
		printer = &runtimeprinters.YAMLPrinter{}
	case OutputFormatJSON:
		printer = &runtimeprinters.JSONPrinter{}
	case OutputFormatName:
		printer = NewNamePrinter()
	default:
		return nil, fmt.Errorf("invalid format %v specified, must be one of: %v",
			output, strings.Join(GetAllOutputFormatStrings(), ","))
	}

	// Don't show managed fields by default for JSON and YAML.
	if output == OutputFormatYAML || output == OutputFormatJSON {
		printer = &runtimeprinters.OmitManagedFieldsPrinter{
			Delegate: printer,
		}
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

// PrintObjects takes the list of items and converts it to a single List, before
// printing it with PrintObject.
func PrintObjects(gvk schema.GroupVersionKind, output OutputFormat, out io.Writer, objs []Object) error {
	var obj Object

	// Set the GVK of all individual items first.
	for _, obj := range objs {
		obj.SetGroupVersionKind(gvk)
	}

	// Convert to a list unless we only have a single object to print.
	if len(objs) == 1 {
		obj = objs[0]
	} else {
		obj = toList(objs)
		gvk = metav1.Unversioned.WithKind("List")
	}

	return PrintObject(gvk, output, out, obj)
}

func toList(objs []Object) *metav1.List {
	// Use a List to print multiple objects instead.
	extensions := make([]runtime.RawExtension, 0, len(objs))
	for _, item := range objs {
		extensions = append(extensions, runtime.RawExtension{
			Object: item,
		})
	}

	return &metav1.List{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.SchemeGroupVersion.Version,
		},
		Items: extensions,
	}
}
