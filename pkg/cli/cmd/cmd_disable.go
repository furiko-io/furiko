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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/furiko-io/furiko/pkg/cli/streams"
)

var (
	DisableExample = PrepareExample(`
# Disable scheduling for the JobConfig.
{{.CommandName}} disable send-weekly-report`)
)

type DisableCommand struct {
	streams *streams.Streams
	name    string
}

func NewDisableCommand(streams *streams.Streams) *cobra.Command {
	c := &DisableCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable automatic scheduling for a JobConfig.",
		Long: `Disables automatic scheduling for a JobConfig.

If the specified JobConfig does not have a schedule, then an error will be thrown.
If the specified JobConfig is already disabled, then this is a no-op.`,
		Example: DisableExample,
		Args:    cobra.ExactArgs(1),
		PreRunE: PrerunWithKubeconfig,
		RunE:    c.Run,
	}

	return cmd
}

func (c *DisableCommand) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client := ctrlContext.Clientsets().Furiko().ExecutionV1alpha1()
	namespace, err := GetNamespace(cmd)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return errors.New("job config name must be specified")
	}
	name := args[0]

	jobConfig, err := client.JobConfigs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot get job config")
	}

	key, err := cache.MetaNamespaceKeyFunc(jobConfig)
	if err != nil {
		return errors.Wrapf(err, "key func error")
	}

	if jobConfig.Spec.Schedule == nil {
		return fmt.Errorf("job config has no schedule specified")
	}
	if jobConfig.Spec.Schedule.Disabled {
		c.streams.Printf("Job config %v is already disabled", key)
		return nil
	}

	newJobConfig := jobConfig.DeepCopy()
	newJobConfig.Spec.Schedule.Disabled = true
	updatedJobConfig, err := client.JobConfigs(namespace).Update(ctx, newJobConfig, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot update job config")
	}
	klog.V(1).InfoS("updated job config", "namespace", updatedJobConfig.Namespace, "name", updatedJobConfig.Name)

	c.streams.Printf("Successfully disabled automatic scheduling for job config %v\n", key)
	return nil
}
