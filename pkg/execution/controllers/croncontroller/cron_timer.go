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
	"math"
	"time"
)

// CronTimerUntil runs work every timestep until stopCh is signaled.
// The work function will first be called immediately on initial call, followed by
// rounding up to the nearest timestep for every subsequent step.
// The smallest timestep that is admissible is a second.
func CronTimerUntil(work func(), timestep time.Duration, stopCh <-chan struct{}) {
	step := timestep.Seconds()
	var nextInterval time.Duration

	for {
		select {
		case <-stopCh:
			return
		case <-Clock.After(nextInterval):
		}

		work()

		// Reset timer to start of next timestep.
		now := Clock.Now()
		nextTS := int64(math.Floor((float64(now.Unix())+step)/step) * step)
		nextInterval = time.Unix(nextTS, 0).Sub(now)
	}
}
