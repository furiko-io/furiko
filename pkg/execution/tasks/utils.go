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

package tasks

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/execution/util/parallel"
	"github.com/furiko-io/furiko/pkg/utils/meta"
)

// SetLabelsAnnotations sets the labels and annotations on a task given a job and task index.
func SetLabelsAnnotations(task meta.LabelableAnnotatable, job *execution.Job, index TaskIndex) error {
	hash, err := parallel.HashIndex(index.Parallel)
	if err != nil {
		return errors.Wrapf(err, "cannot hash parallel index")
	}

	// Set labels.
	meta.SetLabel(task, LabelKeyJobUID, string(job.GetUID()))
	meta.SetLabel(task, LabelKeyTaskRetryIndex, strconv.Itoa(int(index.Retry)))
	meta.SetLabel(task, LabelKeyTaskParallelIndexHash, hash)

	// Compute annotations.
	parallelIndexMarshaled, err := json.Marshal(index.Parallel)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal parallel index")
	}
	meta.SetAnnotation(task, AnnotationKeyTaskParallelIndex, string(parallelIndexMarshaled))

	return nil
}
