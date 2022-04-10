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

package croncontroller

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/furiko-io/furiko/pkg/utils/eventhandler"
)

// InformerWorker receives events from the informer and enqueues work to be done
// for the controller.
type InformerWorker struct {
	*Context
}

func NewInformerWorker(ctrlContext *Context) *InformerWorker {
	w := &InformerWorker{
		Context: ctrlContext,
	}

	// Add event handler when we get JobConfig updates.
	w.jobconfigInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.handleUpdate(oldObj, newObj)
		},
		DeleteFunc: w.enqueueFlush,
	})

	return w
}

func (w *InformerWorker) WorkerName() string {
	return fmt.Sprintf("%v.Informer", controllerName)
}

func (w *InformerWorker) handleUpdate(oldObj, newObj interface{}) {
	oldRjc, err := eventhandler.Executionv1alpha1JobConfig(oldObj)
	if err != nil {
		klog.ErrorS(err, "croncontroller: unable to handle event", "worker", w.WorkerName())
		return
	}
	newRjc, err := eventhandler.Executionv1alpha1JobConfig(newObj)
	if err != nil {
		klog.ErrorS(err, "croncontroller: unable to handle event", "worker", w.WorkerName())
		return
	}

	// Flush next schedule time if its scheduling policy was updated.
	isEqual, err := IsScheduleEqual(oldRjc.Spec.Schedule, newRjc.Spec.Schedule)
	if err != nil {
		klog.ErrorS(err, "croncontroller: cannot compare equal",
			"oldRjc", oldRjc,
			"newRjc", newRjc,
		)
		return
	}

	// Detected update.
	if !isEqual {
		w.enqueueFlush(newRjc)
	}
}

func (w *InformerWorker) enqueueFlush(obj interface{}) {
	rjc, err := eventhandler.Executionv1alpha1JobConfig(obj)
	if err == nil {
		klog.V(5).InfoS("croncontroller: flushing last schedule time of JobConfig",
			"worker", w.WorkerName(),
			"namespace", rjc.GetNamespace(),
			"name", rjc.GetName(),
			"schedule", spew.Sdump(rjc.Spec.Schedule),
		)
		w.UpdatedConfigs <- rjc
	}
}
