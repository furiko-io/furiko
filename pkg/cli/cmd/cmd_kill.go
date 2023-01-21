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
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/furiko-io/furiko/pkg/cli/formatter"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

var (
	KillExample = PrepareExample(`
# Request to kill an ongoing Job.
{{.CommandName}} kill job-sample-1653825000

# Request for an ongoing Job to be killed 60 seconds from now.
{{.CommandName}} kill job-sample-1653825000 -p 60s

# Request for an ongoing Job to be killed at a specific time in the future.
{{.CommandName}} kill job-sample-1653825000 -t 2023-01-01T00:00:00Z`)
)

type KillCommand struct {
	streams   *streams.Streams
	name      string
	override  bool
	killAt    string
	killAfter time.Duration
}

func NewKillCommand(streams *streams.Streams) *cobra.Command {
	c := &KillCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:               "kill",
		Short:             "Kill an ongoing Job.",
		Long:              `Kills an ongoing Job that is currently running or pending.`,
		Example:           KillExample,
		Args:              cobra.ExactArgs(1),
		PreRunE:           PrerunWithKubeconfig,
		ValidArgsFunction: MakeCobraCompletionFunc((&CompletionHelper{}).ListJobs()),
		RunE:              c.Run,
	}

	cmd.Flags().BoolVar(&c.override, "override", false,
		"If the Job already has a kill timestamp, specifying this flag allows overriding the previous value.")
	cmd.Flags().StringVarP(&c.killAt, "at", "t", "",
		"Specify an explicit timestamp to kill the job at, in RFC3339 format.")
	cmd.Flags().DurationVarP(&c.killAfter, "after", "p", 0,
		"Specify a duration relative to the current time that the job should be killed.")

	return cmd
}

func (c *KillCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := GetNamespace(cmd)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return errors.New("job name must be specified")
	}
	name := args[0]

	// Validate flags.
	if c.killAt != "" && c.killAfter != 0 {
		return errors.Wrapf(err, "cannot only specify at most one of: --at, --after")
	}
	if c.killAfter < 0 {
		return fmt.Errorf("must be a positive duration: %v", c.killAfter)
	}

	killAt := ktime.Now()
	if len(c.killAt) > 0 {
		parsed, err := time.Parse(time.RFC3339, c.killAt)
		if err != nil {
			return errors.Wrapf(err, "cannot parse kill timestamp: %v", c.killAt)
		}
		mt := metav1.NewTime(parsed)
		killAt = &mt
	}
	if c.killAfter > 0 {
		mt := metav1.NewTime(ktime.Now().Add(c.killAfter))
		killAt = &mt
	}

	job, err := client.Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot get job")
	}

	// Cannot kill if a kill timestamp is already set, unless --override is specified.
	if !job.Spec.KillTimestamp.IsZero() && !c.override {
		return fmt.Errorf("job already has a kill timestamp set to %v, use --override to override previous value",
			job.Spec.KillTimestamp)
	}

	newJob := job.DeepCopy()
	newJob.Spec.KillTimestamp = killAt

	// Kill the job.
	updatedJob, err := client.Jobs(namespace).Update(ctx, newJob, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot update job")
	}
	klog.V(1).InfoS("updated job", "namespace", updatedJob.Namespace, "name", updatedJob.Name)

	key, err := cache.MetaNamespaceKeyFunc(updatedJob)
	if err != nil {
		return errors.Wrapf(err, "key func error")
	}

	c.streams.Printf("Requested for job %v to be killed at %v\n", key, formatter.FormatTime(killAt))
	return nil
}
