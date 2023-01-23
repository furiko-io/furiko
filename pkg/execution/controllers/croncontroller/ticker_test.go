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

package croncontroller_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	fakeclock "k8s.io/utils/clock/testing"

	"github.com/furiko-io/furiko/pkg/execution/controllers/croncontroller"
)

const (
	realTimeProcessingDelay = time.Millisecond * 20
)

var initialTime = time.Unix(1629701607, 5e8)

func TestClockTickUntil(t *testing.T) {
	var i int64
	work := func() {
		atomic.AddInt64(&i, 1)
	}
	fakeClock := fakeclock.NewFakeClock(initialTime)
	croncontroller.Clock = fakeClock

	step := 5
	timestep := time.Second * time.Duration(step)
	stopCh := make(chan struct{}, 1)
	go croncontroller.ClockTickUntil(work, timestep, stopCh)

	set := func(t time.Time) {
		fakeClock.SetTime(t)
		time.Sleep(realTimeProcessingDelay)
	}

	// Initialise with fake time.
	set(initialTime)

	// Should be called immediately (i.e. 1ns later).
	set(initialTime.Add(time.Nanosecond))
	assert.Equal(t, int64(1), i, "work() was not called immediately")

	// Not called until next timestep.
	set(time.Unix(1629701608, 0))
	assert.Equal(t, int64(1), i, "work() should not be called until start of next timestep")

	// Called right at the beginning of next timestep.
	set(time.Unix(1629701610, 0))
	assert.Equal(t, int64(2), i, "work() should be called immediately at start of next timestep")

	// Not called exactly 5 seconds after start.
	set(initialTime.Add(timestep))
	assert.Equal(t, int64(2), i, "work() should not be called again")

	// Called right at the beginning of next timestep.
	set(time.Unix(1629701615, 0))
	assert.Equal(t, int64(3), i, "work() should be called immediately at start of next timestep")

	// Called right at the beginning of next timestep.
	set(time.Unix(1629701620, 0))
	assert.Equal(t, int64(4), i, "work() should be called immediately at start of next timestep")

	// Jump a few timesteps.
	for i := 1; i <= 4; i++ {
		set(time.Unix(int64(1629701620+i*step), 0))
	}
	assert.Equal(t, int64(8), i, "work() should be called the correct number of times after jumping timesteps")
}

func TestCronTimerUntil_Seconds(t *testing.T) {
	var i int64
	work := func() {
		atomic.AddInt64(&i, 1)
	}
	fakeClock := fakeclock.NewFakeClock(initialTime)
	croncontroller.Clock = fakeClock

	timestep := time.Second
	stopCh := make(chan struct{}, 1)
	go croncontroller.ClockTickUntil(work, timestep, stopCh)

	set := func(t time.Time) {
		fakeClock.SetTime(t)
		time.Sleep(realTimeProcessingDelay)
	}

	// Initialise with fake time.
	set(initialTime)

	// Should be called immediately (i.e. 1ns later).
	set(initialTime.Add(time.Nanosecond))
	assert.Equal(t, int64(1), i, "work() was not called immediately")

	// Not called until next timestep.
	set(initialTime.Add(time.Millisecond * 100))
	assert.Equal(t, int64(1), i, "work() should not be called until start of next timestep")

	// Called right at the beginning of next timestep.
	set(time.Unix(1629701608, 0))
	assert.Equal(t, int64(2), i, "work() should be called immediately at start of next timestep")

	// Not called exactly 1 seconds after start.
	set(initialTime.Add(timestep))
	assert.Equal(t, int64(2), i, "work() should not be called again")

	// Called right at the beginning of next timestep.
	set(time.Unix(1629701609, 0))
	assert.Equal(t, int64(3), i, "work() should be called immediately at start of next timestep")

	// Called right at the beginning of next timestep.
	set(time.Unix(1629701610, 0))
	assert.Equal(t, int64(4), i, "work() should be called immediately at start of next timestep")

	// Jump a few timesteps.
	jump := 10
	for i := 1; i <= jump; i++ {
		set(time.Unix(int64(1629701610+i), 0))
	}
	assert.Equal(t, int64(4+jump), i, "work() should be called the correct number of times after jumping timesteps")
}
