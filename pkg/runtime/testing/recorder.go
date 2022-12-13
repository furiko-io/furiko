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

package testing

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

type Event struct {
	UID     types.UID
	Type    string
	Reason  string
	Message string
}

type uid interface {
	GetUID() types.UID
}

// FakeRecorder implements record.EventRecorder to intercept and compare sent events.
type FakeRecorder struct {
	events []Event
	mu     sync.RWMutex
}

var _ record.EventRecorder = (*FakeRecorder)(nil)

func NewFakeRecorder() *FakeRecorder {
	return &FakeRecorder{}
}

func (f *FakeRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	obj, ok := object.(uid)
	if !ok {
		// no uid, silently drop event
		return
	}
	f.events = append(f.events, Event{
		UID:     obj.GetUID(),
		Type:    eventtype,
		Reason:  reason,
		Message: message,
	})
}

func (f *FakeRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	f.Event(object, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

func (f *FakeRecorder) AnnotatedEventf(
	object runtime.Object,
	annotations map[string]string,
	eventtype, reason, messageFmt string,
	args ...interface{},
) {
	// NOTE(irvinlim): Currently we just ignore annotations
	f.Eventf(object, eventtype, reason, messageFmt, args...)
}

func (f *FakeRecorder) GetEvents() []Event {
	f.mu.RLock()
	defer f.mu.RUnlock()
	events := make([]Event, 0, len(f.events))
	events = append(events, f.events...)
	return events
}
