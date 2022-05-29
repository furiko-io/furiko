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

package parallel

import (
	"encoding/base32"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/utils/matrix"
)

// GenerateIndexes generates the indexes for a ParallelismSpec.
// The order of results is guaranteed to be deterministic.
func GenerateIndexes(spec *execution.ParallelismSpec) []execution.ParallelIndex {
	if spec == nil {
		spec = &execution.ParallelismSpec{}
	}

	switch {
	case spec.WithCount != nil:
		indexes := make([]execution.ParallelIndex, *spec.WithCount)
		for i := int64(0); i < *spec.WithCount; i++ {
			indexes[i] = execution.ParallelIndex{
				IndexNumber: pointer.Int64(i),
			}
		}
		return indexes

	case len(spec.WithKeys) > 0:
		indexes := make([]execution.ParallelIndex, 0, len(spec.WithKeys))
		for _, key := range spec.WithKeys {
			indexes = append(indexes, execution.ParallelIndex{
				IndexKey: key,
			})
		}
		return indexes

	case len(spec.WithMatrix) > 0:
		combinations := matrix.GenerateMatrixCombinations(spec.WithMatrix)
		indexes := make([]execution.ParallelIndex, 0, len(combinations))
		for _, combination := range combinations {
			indexes = append(indexes, execution.ParallelIndex{
				MatrixValues: combination,
			})
		}
		return indexes
	}

	// Default to single count index.
	return []execution.ParallelIndex{GetDefaultIndex()}
}

// GetDefaultIndex returns the default ParallelIndex for a non-parallel job.
func GetDefaultIndex() execution.ParallelIndex {
	return execution.ParallelIndex{
		IndexNumber: pointer.Int64(0),
	}
}

// HashIndex returns a deterministic hash of a ParallelIndex.
// For example, the result of GetDefaultIndex() returns "gezdqo".
func HashIndex(index execution.ParallelIndex) (string, error) {
	hashInt, err := hashstructure.Hash(index, hashstructure.FormatV2, nil)
	if err != nil {
		return "", errors.Wrapf(err, "cannot hash index")
	}
	hash := base32.StdEncoding.EncodeToString([]byte(strconv.FormatUint(hashInt, 10)))
	// NOTE(irvinlim): Use first 6 bytes which should be sufficient for most cases.
	return strings.ToLower(hash[:6]), nil
}

// HashIndexes returns mapping of hashes of ParallelIndex. The first maps the
// slice index to the hash, and the second maps the hash to the slice index.
func HashIndexes(indexes []execution.ParallelIndex) (map[int]string, map[string]int, error) {
	hashes := make(map[int]string, len(indexes))
	hashesIdx := make(map[string]int, len(indexes))
	for i, index := range indexes {
		hash, err := HashIndex(index)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot hash index %v", index)
		}
		hashesIdx[hash] = i
		hashes[i] = hash
	}
	return hashes, hashesIdx, nil
}

// IndexCreationRequest contains an index that should be created, and the earliest time it can be created.
type IndexCreationRequest struct {
	ParallelIndex execution.ParallelIndex
	RetryIndex    int64
	Earliest      time.Time
}

// ComputeMissingIndexesForCreation returns a list of expected indexes based on taskStatuses.
func ComputeMissingIndexesForCreation(
	job *execution.Job,
	indexes []execution.ParallelIndex,
) ([]IndexCreationRequest, error) {
	foundList := make([]bool, len(indexes))
	nextRetryIndex := make(map[string]int64, len(indexes))
	latestFinishTimeByIndex := make(map[string]time.Time, len(indexes))

	// Hash each index, and store a mapping of hash to index to look up later.
	hashes, hashesIdx, err := HashIndexes(indexes)
	if err != nil {
		return nil, err
	}

	// Iterate all existing tasks in the Job's status.
	for _, task := range job.Status.Tasks {
		index := GetDefaultIndex()
		if task.ParallelIndex != nil {
			index = *task.ParallelIndex
		}

		hash, err := HashIndex(index)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot hash index %v", index)
		}

		// Record the maximum retry index and finish time.
		if nextRetryIndex[hash] < task.RetryIndex+1 {
			nextRetryIndex[hash] = task.RetryIndex + 1
		}
		if finish := task.FinishTimestamp; !finish.IsZero() && latestFinishTimeByIndex[hash].Before(finish.Time) {
			latestFinishTimeByIndex[hash] = finish.Time
		}

		// Only handle tasks that are active or successful.
		if task.FinishTimestamp.IsZero() || task.Status.Result == execution.TaskSucceeded {
			foundList[hashesIdx[hash]] = true
		}
	}

	// Extract out all indexes that are not seen.
	requests := make([]IndexCreationRequest, 0, len(indexes))
	for i, found := range foundList {
		index := indexes[i]
		hash := hashes[i]

		// We found an active or successful task.
		if found {
			continue
		}

		// Cannot create because exceeds maxAttempts.
		if nextRetryIndex[hash] >= job.GetMaxAttempts() {
			continue
		}

		// Found a missing index.
		requests = append(requests, IndexCreationRequest{
			ParallelIndex: index,
			RetryIndex:    nextRetryIndex[hash],
			Earliest:      latestFinishTimeByIndex[hash].Add(job.GetRetryDelay()),
		})
	}

	return requests, nil
}
