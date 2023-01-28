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
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	execution "github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/prompt"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/core/options"
	"github.com/furiko-io/furiko/pkg/utils/ktime"
)

var (
	RunExample = PrepareExample(`
# Execute a new Job from an existing JobConfig.
{{.CommandName}} run daily-send-email

# Execute a new Job only after the specified time.
{{.CommandName}} run daily-send-email --at 2021-01-01T00:00:00+08:00

# Execute a new Job only after the specified duration.
{{.CommandName}} run daily-send-email --after 15m

# Execute a new Job, and use all default options.
{{.CommandName}} run daily-send-email --use-default-options

# Execute a new Job with the specified option values in JSON format.
{{.CommandName}} run daily-send-email -O '{"dest_email": "team-leads@listserv.acme.org"}'

# Queue a new Job to be executed after other Jobs are finished.
{{.CommandName}} run process-payments --enqueue`)

	groupResourceJob = execution.Resource("job")
)

type RunCommand struct {
	streams           *streams.Streams
	name              string
	generateName      string
	noInteractive     bool
	optionValues      map[string]interface{}
	useDefaultOptions bool
	startAfter        time.Time
	concurrencyPolicy execution.ConcurrencyPolicy
	displayIntro      sync.Once
}

func NewRunCommand(streams *streams.Streams) *cobra.Command {
	c := &RunCommand{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a new Job.",
		Long: `Runs a new Job from an existing JobConfig.

If the JobConfig has some options defined, an interactive prompt will be shown.`,
		Example:           RunExample,
		Args:              cobra.ExactArgs(1),
		PreRunE:           PrerunWithKubeconfig,
		ValidArgsFunction: MakeCobraCompletionFunc((&CompletionHelper{}).ListJobConfigs()),
		RunE: RunAllE(
			c.Complete,
			c.Validate,
			c.Run,
		),
	}

	cmd.Flags().StringVar(&c.name, "name", "",
		"Specifies a name to use for the created Job, otherwise it will be generated based on the job config's name.")
	cmd.Flags().StringVar(&c.generateName, "generate-name", "",
		"Specifies a name prefix (i.e. generateName) to use for the created Job, otherwise it defaults to the job config's name.")
	cmd.Flags().BoolVar(&c.noInteractive, "no-interactive", false,
		"If specified, will not show an interactive prompt. This may result in an error when certain values are "+
			"required but not provided.")
	cmd.Flags().StringP("option-values", "O", "",
		"Option values to be used for executing the job, in JSON format. "+
			"Commonly used in conjunction with --no-interactive in order to execute a new job without prompts. "+
			"If --use-default-options is also specified, any matching keys in --option-values will override the default option values.")
	cmd.Flags().BoolVar(&c.useDefaultOptions, "use-default-options", false,
		"If specified, options with default values defined and will not show an interactive prompt. "+
			"Any options without default values will still show one, unless --no-interactive is set.")
	cmd.Flags().String("at", "",
		"RFC3339-formatted datetime to specify the time to run the job at. "+
			"Implies --concurrency-policy=Enqueue unless explicitly specified.")
	cmd.Flags().String("after", "",
		"Duration from current time that the job should be scheduled to be started at. "+
			"Must be formatted as a Golang duration string, e.g. 5m, 3h, etc. "+
			"Shorthand for --at=[now() + after]. Implies --concurrency-policy=Enqueue unless explicitly specified.")
	cmd.Flags().String("concurrency-policy", "",
		"Specify an explicit concurrency policy to use for the job, overriding the "+
			"JobConfig's concurrency policy.")
	cmd.Flags().Bool("enqueue", false,
		"Enqueues the job to be executed, which will only start after other ongoing jobs. "+
			"Shorthand for --concurrency-policy=Enqueue.")

	if err := RegisterFlagCompletions(cmd, []FlagCompletion{
		{FlagName: "concurrency-policy", CompletionFunc: (&CompletionHelper{}).FromSlice(execution.ConcurrencyPoliciesAll)},
	}); err != nil {
		Fatal(err, DefaultErrorExitCode)
	}

	return cmd
}

func (c *RunCommand) Complete(cmd *cobra.Command, args []string) error {
	// Handle --concurrency-policy.
	if concurrencyPolicy := GetFlagString(cmd, "concurrency-policy"); concurrencyPolicy != "" {
		c.concurrencyPolicy = execution.ConcurrencyPolicy(concurrencyPolicy)
	}

	// Handle --enqueue shorthand flag.
	if GetFlagBool(cmd, "enqueue") {
		if c.concurrencyPolicy != "" {
			return fmt.Errorf("cannot specify both --enqueue and --concurrency-policy together")
		}
		c.concurrencyPolicy = execution.ConcurrencyPolicyEnqueue
	}

	// Parse --at as timestamp.
	if at := GetFlagString(cmd, "at"); at != "" {
		parsed, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return errors.Wrapf(err, "invalid value for --at: cannot parse %v as RFC3339 timestamp", c.startAfter)
		}
		c.startAfter = parsed
	}

	// Handle --after shorthand flag.
	if after := GetFlagString(cmd, "after"); after != "" {
		if !c.startAfter.IsZero() {
			return fmt.Errorf("cannot specify both --after and --at together")
		}
		duration, err := time.ParseDuration(after)
		if err != nil {
			return errors.Wrapf(err, "invalid value for --after: cannot parse %v as duration", after)
		}
		c.startAfter = ktime.Now().Add(duration)
	}

	// Use Enqueue if --after or --at is specified by default.
	if !c.startAfter.IsZero() && c.concurrencyPolicy == "" {
		c.concurrencyPolicy = execution.ConcurrencyPolicyEnqueue
	}

	// Handle --option-values.
	if optionValues := GetFlagString(cmd, "option-values"); optionValues != "" {
		if err := json.Unmarshal([]byte(optionValues), &c.optionValues); err != nil {
			return errors.Wrapf(err, "invalid value for --option-values: cannot unmarshal as JSON")
		}
	}

	return nil
}

func (c *RunCommand) Validate(cmd *cobra.Command, args []string) error {
	// Both --name and --generate-name cannot be specified together.
	if c.name != "" && c.generateName != "" {
		return fmt.Errorf("cannot specify both --name and --generate-name together")
	}

	// Validate --concurrency-policy.
	if c.concurrencyPolicy != "" && !c.concurrencyPolicy.IsValid() {
		return fmt.Errorf("invalid value for --concurrency-policy, valid values: %v", execution.ConcurrencyPoliciesAll)
	}

	return nil
}

func (c *RunCommand) Run(cmd *cobra.Command, args []string) error {
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
			Name:         c.name,
			GenerateName: c.generateName,
			Namespace:    jobConfig.Namespace,
		},
		Spec: execution.JobSpec{
			ConfigName:   name,
			StartPolicy:  startPolicy,
			OptionValues: optionValues,
		},
	}

	// Generate name from JobConfig's name if not specified.
	if newJob.Name == "" && newJob.GenerateName == "" {
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

	c.streams.Printf("Job %v created\n", key)
	return nil
}

func (c *RunCommand) makeJobStartPolicy() (*execution.StartPolicySpec, error) {
	startPolicy := &execution.StartPolicySpec{
		ConcurrencyPolicy: c.concurrencyPolicy,
	}

	// Set startAfter.
	if !c.startAfter.IsZero() {
		startAfterTime := metav1.NewTime(c.startAfter)
		startPolicy.StartAfter = &startAfterTime
	}

	return startPolicy, nil
}

func (c *RunCommand) makeJobOptionValues(jobConfig *execution.JobConfig) (string, error) {
	if jobConfig.Spec.Option == nil {
		return "", nil
	}

	values := make(map[string]interface{}, len(jobConfig.Spec.Option.Options))
	for _, option := range jobConfig.Spec.Option.Options {
		value, err := c.makeJobOptionValue(option)
		if err != nil {
			return "", err
		}
		if value != nil {
			values[option.Name] = value
		}
	}

	// Marshal the result.
	var result string
	if len(values) > 0 {
		marshaled, err := json.Marshal(values)
		if err != nil {
			return "", errors.Wrapf(err, "marshal error")
		}
		klog.V(2).InfoS("evaluated option values", "optionValues", string(marshaled))
		result = string(marshaled)
	}

	return result, nil
}

func (c *RunCommand) makeJobOptionValue(option execution.Option) (interface{}, error) {
	// If the option is already defined via --option-values, use the value instead.
	// This helps to suppress unnecessary prompts.
	if value, ok := c.optionValues[option.Name]; ok {
		return value, nil
	}

	// If flag is defined to use default options, first check if we can use default value.
	var hasDefaultValue bool
	if c.useDefaultOptions {
		ok, err := c.checkOptionValueHasDefault(option)
		if err != nil {
			return nil, err
		}
		hasDefaultValue = ok
	}

	var value interface{}

	// Don't need to display interactive prompt if we got a default value just now.
	// However, we also don't display the prompt if the user specifies
	// --no-interactive (e.g. for use in scripts).
	if !hasDefaultValue && !c.noInteractive {
		c.displayIntro.Do(func() { color.HiBlack("Please input option values.\n") })
		prompter, err := prompt.MakePrompt(c.streams, option)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot make prompt for option: %v", option.Name)
		}
		newValue, err := prompter.Run()
		if err != nil {
			return nil, errors.Wrapf(err, "prompt error")
		}
		klog.V(4).Infof(`prompt evaluated option "%v" value: "%v"`, option.Name, value)
		value = newValue
	}

	return value, nil
}

func (c *RunCommand) checkOptionValueHasDefault(option execution.Option) (bool, error) {
	// Get default value.
	defaultValue, err := options.GetOptionDefaultValue(option)
	if err != nil {
		return false, errors.Wrapf(err, `cannot get default value for option "%v"`, option.Name)
	}

	// Evaluate the option, and check if a Required error is thrown.
	// Each option may handle Required checking differently.
	_, fieldErr := options.EvaluateOption(defaultValue, option, field.NewPath(""))
	if fieldErr != nil && fieldErr.Type != field.ErrorTypeRequired {
		return false, errors.Wrapf(err, `cannot evaluate option "%v"`, option.Name)
	}

	return fieldErr == nil, nil
}
