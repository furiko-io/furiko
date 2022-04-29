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

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
)

// NamePrinter is an implementation of ResourcePrinter which outputs
// "resource/name" pair of an object. This printer extends the default
// printers.NamePrinter to work with lists by assuming that all items have a
// GroupVersionKind already set.
type NamePrinter struct {
	printer *printers.NamePrinter
}

func NewNamePrinter() *NamePrinter {
	return &NamePrinter{
		printer: &printers.NamePrinter{},
	}
}

var _ printers.ResourcePrinter = (*NamePrinter)(nil)

func (p *NamePrinter) PrintObj(obj runtime.Object, w io.Writer) error {
	if meta.IsListType(obj) {
		items, err := meta.ExtractList(obj)
		if err != nil {
			return err
		}
		for _, item := range items {
			obj, ok := item.(Object)
			if !ok {
				return fmt.Errorf("item is not an ObjectKind")
			}
			if err := p.PrintObj(obj, w); err != nil {
				return err
			}
		}

		return nil
	}

	return p.printer.PrintObj(obj, w)
}
