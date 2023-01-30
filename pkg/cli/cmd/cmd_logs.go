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

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/execution/util/parallel"
)

var (
	LogsExample = common.PrepareExample(`
# Load the logs for the latest task attempt of the given Job.
{{.CommandName}} logs job-sample-1653825000`)
)

type LogsCommand struct {
	streams                      *streams.Streams
	parallelIndex                *execution.ParallelIndex
	retryIndex                   *int64
	containerName                string
	follow                       bool
	previous                     bool
	limitBytes                   *int64
	tailLines                    *int64
	insecureSkipTLSVerifyBackend bool
}

func NewLogsCommand(streams *streams.Streams) *cobra.Command {
	c := &LogsCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:               "logs",
		Short:             "Load and stream logs for a Job's pod.",
		Long:              `Loads and streams logs for a Job's pod. By default, the latest retry attempt's pod logs will be loaded.'`,
		Example:           LogsExample,
		Args:              cobra.ExactArgs(1),
		PreRunE:           common.PrerunWithKubeconfig,
		ValidArgsFunction: completion.CompleterToCobraCompletionFunc(&completion.ListJobsCompleter{Filter: jobutil.IsStarted}),
		RunE: common.RunAllE(
			c.Complete,
			c.Validate,
			c.Run,
		),
	}

	// Common flags for PodLogOptions.
	cmd.Flags().StringVarP(&c.containerName, "container", "c", "",
		"Print the logs of this container.")
	cmd.Flags().BoolVarP(&c.follow, "follow", "f", false,
		"Specify if the logs should be streamed.")
	cmd.Flags().BoolVarP(&c.previous, "previous", "p", false,
		"If true, print the logs for the previous instance of the container in a pod if it exists.")
	cmd.Flags().Int64("limit-bytes", 0,
		"Maximum bytes of logs to return. Defaults to no limit.")
	cmd.Flags().Int64P("tail", "t", -1,
		"Lines of recent log file to display. Defaults to -1, showing all log lines.")
	cmd.Flags().BoolVar(&c.insecureSkipTLSVerifyBackend, "insecure-skip-tls-verify-backend", false,
		"Skip verifying the identity of the kubelet that logs are requested from.\n"+
			"In theory, an attacker could provide invalid log content back.\n"+
			"You might want to use this if your kubelet serving certificates have expired.")

	// Flags for task selection.
	cmd.Flags().StringP("index", "i", "",
		"Specifies the parallel index, which determines which task to load logs for.\n"+
			"If the Job is not parallel, this flag can be omitted and defaults to the only task.\n"+
			"Must be specified as one of the following: \n"+
			"	INT (indexNumber)\n"+
			"	STRING (indexKey)\n"+
			"	KEY=VALUE[,KEY2=VALUE2] (matrixValues)\n"+
			`	{"indexNumber": 0} (JSON representation)`)
	cmd.Flags().StringP("retry-index", "r", "",
		"Specify which retry attempt should be loaded. By default, the latest retry attempt will be used.")

	return cmd
}

func (c *LogsCommand) Complete(cmd *cobra.Command, args []string) error {
	if limitBytes := common.GetFlagInt64(cmd, "limit-bytes"); limitBytes > 0 {
		c.limitBytes = pointer.Int64(limitBytes)
	}
	if tailLines := common.GetFlagInt64(cmd, "tail"); tailLines > 0 {
		c.tailLines = pointer.Int64(tailLines)
	}

	// Handle --index.
	if index := common.GetFlagString(cmd, "index"); index != "" {
		parallelIndex, err := ParseParallelIndex(index)
		if err != nil {
			return errors.Wrapf(err, "invalid value for --index: cannot parse %v", index)
		}
		c.parallelIndex = parallelIndex
	}

	// Handle --retry-index.
	if retryIndex := common.GetFlagString(cmd, "retry-index"); retryIndex != "" {
		// Attempt to parse as int.
		index, err := strconv.Atoi(retryIndex)
		if err != nil {
			return errors.Wrapf(err, "invalid value for --retry-index: cannot parse %v as int", retryIndex)
		}
		c.retryIndex = pointer.Int64(int64(index))
	}

	return nil
}

func (c *LogsCommand) Validate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("job name must be specified")
	}

	// Parallel index must be valid.
	if idx := c.parallelIndex; idx != nil {
		var found int
		if idx.IndexNumber != nil {
			found++
		}
		if idx.IndexKey != "" {
			found++
		}
		if len(idx.MatrixValues) > 0 {
			found++
		}
		if found != 1 {
			return fmt.Errorf("invalid value for --index: exactly one of (indexNumber, indexKey, matrixValues) must be defined, got %v kinds", found)
		}
	}

	// Retry index cannot be less than 0.
	if c.retryIndex != nil && *c.retryIndex < 0 {
		return fmt.Errorf("invalid retry index, must be greater or equal to 0")
	}

	return nil
}

func (c *LogsCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	kubeClient := common.GetCtrlContext().Clientsets().Kubernetes().CoreV1()
	executionClient := common.GetCtrlContext().Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := common.GetNamespace(cmd)
	if err != nil {
		return err
	}

	name := args[0]

	// Fetch the job.
	job, err := executionClient.Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot get job")
	}

	// Select a single task to print.
	taskName, err := c.selectTask(job)
	if err != nil {
		return errors.Wrapf(err, "cannot load logs for %v", job.Name)
	}

	// Cannot automatically select a task, inform the user to use --index.
	if taskName == "" {
		return fmt.Errorf("multiple tasks found for %v, use --index to select a specific task to load", job.Name)
	}

	// Fetch the pod logs.
	stream, err := kubeClient.Pods(job.Namespace).GetLogs(taskName, &corev1.PodLogOptions{
		Container:                    c.containerName,
		Follow:                       c.follow,
		Previous:                     c.previous,
		TailLines:                    c.tailLines,
		LimitBytes:                   c.limitBytes,
		InsecureSkipTLSVerifyBackend: c.insecureSkipTLSVerifyBackend,
	}).Stream(ctx)
	if err != nil {
		return errors.Wrapf(err, "cannot stream pod logs for %v", taskName)
	}
	defer stream.Close()

	// Mirror all output from the stream into our output stream.
	if _, err := io.Copy(c.streams.Out, stream); err != nil && err != io.EOF {
		return errors.Wrapf(err, "stream error")
	}

	return nil
}

func (c *LogsCommand) selectTask(job *execution.Job) (string, error) {
	// Job has no created tasks.
	if len(job.Status.Tasks) == 0 {
		return "", fmt.Errorf("no tasks created yet, try again later")
	}

	jobTemplate := job.Spec.Template
	if jobTemplate == nil {
		jobTemplate = &execution.JobTemplate{}
	}

	// Group tasks by parallel index.
	// We want to find the latest started/terminated task for each parallel index,
	// and if there are multiple parallel indexes with such latest task,
	// then we need to perform some selection.
	indexes := parallel.GenerateIndexes(jobTemplate.Parallelism)
	tasks := make(map[string]map[int64]*execution.TaskRef, len(indexes))
	for _, task := range job.Status.Tasks {
		index := parallel.GetDefaultIndex()
		if task.ParallelIndex != nil {
			index = *task.ParallelIndex
		}

		hash, err := parallel.HashIndex(index)
		if err != nil {
			return "", errors.Wrapf(err, "cannot hash index: %v", index)
		}

		// Cannot select tasks that are not yet running
		if task.Status.State == execution.TaskStarting {
			continue
		}

		// Store the task in the map.
		if tasks[hash] == nil {
			tasks[hash] = make(map[int64]*execution.TaskRef)
		}
		tasks[hash][task.RetryIndex] = task.DeepCopy()
	}

	// Use the parallel index if explicitly defined.
	if c.parallelIndex != nil {
		hash, err := parallel.HashIndex(*c.parallelIndex)
		if err != nil {
			return "", errors.Wrapf(err, "cannot hash index: %v", c.parallelIndex)
		}
		retries, ok := tasks[hash]
		if !ok || len(retries) == 0 {
			return "", fmt.Errorf(`no started tasks found for "%v-%v-"`, job.Name, hash)
		}

		task, ok := c.selectRetryIndex(retries)
		if !ok {
			return "", fmt.Errorf(`no started tasks found for "%v-%v-" for retry %v`, job.Name, hash, *c.retryIndex)
		}
		return task.Name, nil
	}

	// No tasks found.
	if len(tasks) == 0 {
		return "", fmt.Errorf("no tasks started running yet, try again later")
	}

	// If the Job is not a parallel job, simply select the only index.
	if jobTemplate.Parallelism == nil && len(tasks) == 1 {
		for _, retries := range tasks {
			task, ok := c.selectRetryIndex(retries)
			if !ok {
				return "", fmt.Errorf("no started tasks found for retry %v", *c.retryIndex)
			}
			return task.Name, nil
		}
	}

	// Otherwise, we are unable to automatically select a task.
	return "", nil
}

// selectRetryIndex returns TaskRef with the maximum index in the retries map.
// If --retry-index is specified, it will attempt to index the map, return false if such index does not exist.
func (c *LogsCommand) selectRetryIndex(retries map[int64]*execution.TaskRef) (*execution.TaskRef, bool) {
	// Use --retry-index.
	if c.retryIndex != nil {
		task, ok := retries[*c.retryIndex]
		if !ok {
			return nil, false
		}
		return task, true
	}

	// Otherwise, use the maximum retry index.
	var maxRetryIndex int64 = -1
	for retryIndex := range retries {
		if retryIndex > maxRetryIndex {
			maxRetryIndex = retryIndex
		}
	}
	return retries[maxRetryIndex], true
}

// ParseParallelIndex is a best-effort attempt to parse a human-readable parallel index string.
func ParseParallelIndex(s string) (*execution.ParallelIndex, error) {
	// Cannot parse empty string.
	if s == "" {
		return nil, fmt.Errorf("cannot parse empty string as parallel index")
	}

	// First try to parse as JSON.
	var index execution.ParallelIndex
	if err := json.Unmarshal([]byte(s), &index); err == nil {
		return &index, nil
	}

	// Try to parse as int for IndexNumber.
	if v, err := strconv.Atoi(s); err == nil {
		return &execution.ParallelIndex{
			IndexNumber: pointer.Int64(int64(v)),
		}, nil
	}

	// If it doesn't contain "=", then it should be IndexKey.
	if !strings.Contains(s, "=") {
		return &execution.ParallelIndex{
			IndexKey: s,
		}, nil
	}

	// Split by commas and equals.
	index = execution.ParallelIndex{
		MatrixValues: map[string]string{},
	}
	for _, kv := range strings.Split(s, ",") {
		toks := strings.SplitN(kv, "=", 2)
		index.MatrixValues[toks[0]] = toks[1]
	}

	return &index, nil
}
