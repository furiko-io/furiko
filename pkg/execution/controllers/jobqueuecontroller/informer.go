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

package jobqueuecontroller

import (
	"fmt"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/furiko-io/furiko/pkg/utils/eventhandler"
	"github.com/furiko-io/furiko/pkg/utils/execution/jobconfig"
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

	// Add event handler for Jobs.
	// We will sync their parent JobConfigs.
	w.jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: w.handleJob,
		UpdateFunc: func(_, newObj interface{}) {
			w.handleJob(newObj)
		},
		DeleteFunc: w.handleJob,
	})

	return w
}

func (w *InformerWorker) WorkerName() string {
	return fmt.Sprintf("%v.Informer", controllerName)
}

// enqueueObject enqueues an object to the workqueue.
// This method also accepts DeletionFinalStateUnknown tombstone objects also since it uses a wrapped KeyFunc.
func (w *InformerWorker) enqueueObject(obj interface{}, queue workqueue.Interface) {
	// Get key to enqueue.
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.ErrorS(err, "jobqueuecontroller: keyfunc error", "worker", w.WorkerName(), "obj", obj)
		return
	}

	// Add to workqueue.
	queue.Add(key)
}

func (w *InformerWorker) handleJob(obj interface{}) {
	rj, err := eventhandler.Executionv1alpha1Job(obj)
	if err != nil {
		klog.ErrorS(err, "jobqueuecontroller: unable to handle event", "worker", w.WorkerName())
		return
	}

	rjc, err := jobconfig.LookupJobOwner(rj, w.jobconfigInformer.Lister().JobConfigs(rj.Namespace))
	if err != nil {
		klog.ErrorS(err, "jobqueuecontroller: cannot look up jobconfig for job",
			"worker", w.WorkerName(),
			"namespace", rj.GetNamespace(),
			"name", rj.GetName(),
		)
		return
	}

	// Enqueue JobConfig to be reconciled.
	if rjc != nil {
		w.enqueueObject(rjc, w.jobConfigQueue)
		return
	}

	// Otherwise, enqueue the Job to be reconciled independently.
	w.enqueueObject(rj, w.independentQueue)
}
