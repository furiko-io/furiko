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
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/util/prompt"
	"github.com/furiko-io/furiko/pkg/core/options"
)

var (
	groupResourceJob = execution.Resource("job")
)

type RunCommand struct {
	name              string
	noInteractive     bool
	useDefaultOptions bool
	startAfter        string
	concurrencyPolicy string
}

func NewRunCommand(ctx context.Context) *cobra.Command {
	c := &RunCommand{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a new Job.",
		Long: `Runs a new Job from an existing JobConfig.

If the JobConfig has some options defined, an interactive prompt will be shown.`,
		Example: `  # Start a new Job from an existing JobConfig.
  furiko run daily-send-email

  # Start a new Job only after the specified time.
  furiko run daily-send-email --at 2021-01-01T00:00:00+08:00

  # Start a new Job, and use all default options.
  furiko run daily-send-email --use-default-options`,
		Args:    cobra.ExactArgs(1),
		PreRunE: PrerunWithKubeconfig,
		RunE:    ToRunE(ctx, c),
	}

	cmd.Flags().StringVar(&c.name, "name", "",
		"Specifies a name to use for the created Job, otherwise it will be generated based on the job config's name.")
	cmd.Flags().BoolVar(&c.noInteractive, "no-interactive", false,
		"If specified, will not show an interactive  prompt. This may result in an error when certain values are "+
			"required but not provided.")
	cmd.Flags().BoolVar(&c.useDefaultOptions, "use-default-options", false,
		"If specified, options with default values defined and will not show an interactive prompt. "+
			"Any options without default values will still show one, unless --no-interactive is set.")
	cmd.Flags().StringVar(&c.startAfter, "at", "",
		"RFC3339-formatted datetime to specify the time to run the job at. "+
			"Implies --concurrency-policy=Enqueue unless explicitly specified.")
	cmd.Flags().StringVar(&c.concurrencyPolicy, "concurrency-policy", "",
		"Specify an explicit concurrency policy to use for the job, overriding the JobConfig's concurrency policy.")

	return cmd
}

func (c *RunCommand) Run(ctx context.Context, cmd *cobra.Command, args []string) error {
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

	// Prepare fields.
	optionValues, err := c.makeJobOptionValues(jobConfig)
	if err != nil {
		return errors.Wrapf(err, "cannot prepare job option values")
	}
	startPolicy, err := c.makeJobStartPolicy()
	if err != nil {
		return errors.Wrapf(err, "cannot prepare start policy")
	}

	// Create a new job using configName.
	newJob := &execution.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.name,
			Namespace: jobConfig.Namespace,
		},
		Spec: execution.JobSpec{
			ConfigName:   name,
			StartPolicy:  startPolicy,
			OptionValues: optionValues,
		},
	}

	// Generate name from JobConfig's name if not specified./
	if newJob.Name == "" {
		newJob.GenerateName = name + "-"
	}

	if klog.V(1).Enabled() {
		if marshaled, err := json.Marshal(newJob); err == nil {
			klog.V(1).InfoS(fmt.Sprintf(`creating %v`, groupResourceJob), "object", marshaled)
		}
	}

	// Submit the job.
	createdJob, err := client.Jobs(namespace).Create(ctx, newJob, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot create job")
	}
	klog.V(1).InfoS("created job", "namespace", createdJob.Namespace, "name", createdJob.Name)

	key, err := cache.MetaNamespaceKeyFunc(createdJob)
	if err != nil {
		return errors.Wrapf(err, "key func error")
	}

	fmt.Printf("Job %v created\n", key)
	return nil
}

func (c *RunCommand) makeJobStartPolicy() (*execution.StartPolicySpec, error) {
	startPolicy := &execution.StartPolicySpec{}

	// Set concurrencyPolicy.
	if c.concurrencyPolicy != "" {
		startPolicy.ConcurrencyPolicy = execution.ConcurrencyPolicy(c.concurrencyPolicy)
	}

	// Set startAfter.
	if c.startAfter != "" {
		parsed, err := time.Parse(time.RFC3339, c.startAfter)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid time: %v", c.startAfter)
		}
		startAfterTime := metav1.NewTime(parsed)
		startPolicy.StartAfter = &startAfterTime

		// Use Enqueue when using --at by default.
		if startPolicy.ConcurrencyPolicy == "" {
			startPolicy.ConcurrencyPolicy = execution.ConcurrencyPolicyEnqueue
		}
	}

	return startPolicy, nil
}

func (c *RunCommand) makeJobOptionValues(jobConfig *execution.JobConfig) (string, error) {
	if jobConfig.Spec.Option == nil {
		return "", nil
	}

	var displayedIntro bool
	values := make(map[string]interface{}, len(jobConfig.Spec.Option.Options))
	for _, option := range jobConfig.Spec.Option.Options {
		var value interface{}

		// Use the default value if defined by flag.
		if c.useDefaultOptions {
			defaultValue, err := options.GetOptionDefaultValue(option)
			if err != nil {
				return "", errors.Wrapf(err, `cannot get default value for option "%v"`, option.Name)
			}
			value = defaultValue
			klog.V(4).Infof(`using default value for option "%v": %v`, option.Name, value)
		}

		// Display interactive prompt unless omitted by user.
		if value == nil && !c.noInteractive {
			// Display introductory message once.
			if !displayedIntro {
				color.HiBlack("Please input option values.\n")
				displayedIntro = true
			}

			prompter, err := prompt.MakePrompt(option)
			if err != nil {
				return "", errors.Wrapf(err, "cannot make prompt for option: %v", option.Name)
			}
			newValue, err := prompter.Run()
			if err != nil {
				return "", errors.Wrapf(err, "prompt error")
			}

			klog.V(4).Infof(`prompt evaluated option "%v" value: "%v"`, option.Name, value)
			value = newValue
		}

		// Finally, add to the map only if non-nil.
		if value != nil {
			values[option.Name] = value
		}
	}

	marshaled, err := json.Marshal(values)
	if err != nil {
		return "", errors.Wrapf(err, "marshal error")
	}
	klog.V(2).InfoS("evaluated option values", "optionValues", string(marshaled))

	return string(marshaled), nil
}
