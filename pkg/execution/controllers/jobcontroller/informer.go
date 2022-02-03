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

package jobcontroller

import (
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
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

	// Add event handler for Pods.
	w.podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: w.handlePod,
		UpdateFunc: func(_, newObj interface{}) {
			w.handlePod(newObj)
		},
		DeleteFunc: w.handlePod,
	})

	w.jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: w.enqueueObject,
		UpdateFunc: func(_, newObj interface{}) {
			w.enqueueObject(newObj)
		},
		DeleteFunc: w.enqueueObject,
	})

	return w
}

func (w *InformerWorker) WorkerName() string {
	return fmt.Sprintf("%v.Informer", controllerName)
}

// enqueueObject enqueues an object to the workqueue.
// This method also accepts DeletionFinalStateUnknown tombstone objects also since it uses a wrapped KeyFunc.
func (w *InformerWorker) enqueueObject(obj interface{}) {
	// Get key to enqueue.
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.ErrorS(err, "jobcontroller: keyfunc error", "worker", w.WorkerName(), "obj", obj)
		return
	}

	// Add to workqueue.
	w.queue.Add(key)
}

func (w *InformerWorker) handlePod(obj interface{}) {
	pod, err := eventhandler.Corev1Pod(obj)
	if err != nil {
		klog.ErrorS(err, "jobcontroller: unable to handle event", "worker", w.WorkerName())
		return
	}

	if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
		rj := w.resolveRefedJob(pod.GetNamespace(), controllerRef)
		if rj != nil {
			w.enqueueObject(rj)
			return
		}
	}
}

// resolveRefedJob returns the Job referenced by ControllerRef.
// It does sanity checks to ensure that we don't return inaccurate objects.
func (w *InformerWorker) resolveRefedJob(namespace string, ref *metav1.OwnerReference) *execution.Job {
	// Wrong kind.
	if ref.Kind != execution.KindJob {
		return nil
	}

	// Look up Job by name.
	rj, err := w.jobInformer.Lister().Jobs(namespace).Get(ref.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			utilruntime.HandleError(err)
		}
		return nil
	}

	// Check if UID matches
	if rj.UID != ref.UID {
		return nil
	}

	return rj
}
