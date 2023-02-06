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

	"github.com/furiko-io/furiko/pkg/cli/common"
	"github.com/furiko-io/furiko/pkg/cli/completion"
	"github.com/furiko-io/furiko/pkg/cli/format"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	jobutil "github.com/furiko-io/furiko/pkg/execution/util/job"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

var (
	KillExample = common.PrepareExample(`
# Request to kill an ongoing Job.
{{.CommandName}} kill job-sample-1653825000

# Request for an ongoing Job to be killed 60 seconds from now.
{{.CommandName}} kill job-sample-1653825000 -p 60s

# Request for an ongoing Job to be killed at a specific time in the future.
{{.CommandName}} kill job-sample-1653825000 -t 2023-01-01T00:00:00Z`)
)

type KillCommand struct {
	streams  *streams.Streams
	name     string
	override bool
	killAt   time.Time
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
		PreRunE:           common.PrerunWithKubeconfig,
		ValidArgsFunction: completion.CompleterToCobraCompletionFunc(&completion.ListJobsCompleter{Filter: jobutil.IsActive}),
		RunE: common.RunAllE(
			c.Complete,
			c.Validate,
			c.Run,
		),
	}

	cmd.Flags().BoolVar(&c.override, "override", false,
		"If the Job already has a kill timestamp, specifying this flag allows overriding the previous value.")
	cmd.Flags().StringP("at", "t", "",
		"Specify an explicit timestamp to kill the job at, in RFC3339 format.")
	cmd.Flags().StringP("after", "p", "",
		"Specify a duration relative to the current time that the job should be killed.")

	return cmd
}

func (c *KillCommand) Complete(cmd *cobra.Command, args []string) error {
	// Parse --at as timestamp.
	if at := common.GetFlagString(cmd, "at"); at != "" {
		parsed, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return errors.Wrapf(err, "invalid value for --at: cannot parse %v as RFC3339 timestamp", at)
		}
		c.killAt = parsed
	}

	// Handle --after shorthand flag.
	if after := common.GetFlagString(cmd, "after"); after != "" {
		if !c.killAt.IsZero() {
			return fmt.Errorf("cannot specify both --after and --at together")
		}
		duration, err := time.ParseDuration(after)
		if err != nil {
			return errors.Wrapf(err, "invalid value for --after: cannot parse %v as duration", after)
		}
		c.killAt = ktime.Now().Add(duration)
	}

	return nil
}

func (c *KillCommand) Validate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("job name must be specified")
	}

	// Don't allow to specify --at with time in the past.
	if !c.killAt.IsZero() && c.killAt.Before(ktime.Now().Time) {
		return fmt.Errorf("cannot specify to kill job after a time in the past: %v", c.killAt)
	}

	return nil
}

func (c *KillCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client := common.GetCtrlContext().Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := common.GetNamespace(cmd)
	if err != nil {
		return err
	}

	name := args[0]

	killAt := ktime.Now()
	if !c.killAt.IsZero() {
		mt := metav1.NewTime(c.killAt)
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

	c.streams.Printf("Requested for job %v to be killed at %v\n", key, format.Time(killAt))
	return nil
}
