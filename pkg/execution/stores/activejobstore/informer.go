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

package activejobstore

import (
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	executioninformers "github.com/furiko-io/furiko/pkg/generated/informers/externalversions/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
	"github.com/furiko-io/furiko/pkg/utils/eventhandler"
)

type EventHandler interface {
	OnUpdate(oldRj, newRj *execution.Job)
	OnDelete(rj *execution.Job)
}

// InformerWorker is used by the store internally to receive updates to the
// central store.
//
// We only listen to Update and Delete events. This is because any additions
// are expected to be done explicitly.
//
// It is acceptable for Update and Delete events to be delayed (but not missed).
type InformerWorker struct {
	controllercontext.Context
	jobInformer executioninformers.JobInformer
	handler     EventHandler
}

func NewInformerWorker(ctrlContext controllercontext.Context, handler EventHandler) *InformerWorker {
	return &InformerWorker{
		Context:     ctrlContext,
		jobInformer: ctrlContext.Informers().Furiko().Execution().V1alpha1().Jobs(),
		handler:     handler,
	}
}

func (w *InformerWorker) Start(stopCh <-chan struct{}) {
	// Add event handlers.
	w.jobInformer.Informer().AddEventHandler(w)

	// Start informers.
	w.Informers().Furiko().Start(stopCh)
}

func (w *InformerWorker) OnAdd(obj interface{}) {
	// Do nothing, as we expect additions to store will be called explicitly (rather than reconciled).
}

func (w *InformerWorker) OnUpdate(oldObj, newObj interface{}) {
	oldRj, err := eventhandler.Executionv1alpha1Job(oldObj)
	if err != nil {
		klog.ErrorS(err, "activejobstore: unable to handle event", "store", storeName)
		return
	}

	newRj, err := eventhandler.Executionv1alpha1Job(newObj)
	if err != nil {
		klog.ErrorS(err, "activejobstore: unable to handle event", "store", storeName)
		return
	}

	w.handler.OnUpdate(oldRj, newRj)
}

func (w *InformerWorker) OnDelete(obj interface{}) {
	rj, err := eventhandler.Executionv1alpha1Job(obj)
	if err != nil {
		klog.ErrorS(err, "activejobstore: unable to handle event", "store", storeName)
		return
	}

	w.handler.OnDelete(rj)
}

func (w *InformerWorker) HasSynced() bool {
	return w.jobInformer.Informer().HasSynced()
}
