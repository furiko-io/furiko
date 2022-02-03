//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BootstrapConfigSpec) DeepCopyInto(out *BootstrapConfigSpec) {
	*out = *in
	out.DefaultResync = in.DefaultResync
	if in.DynamicConfigs != nil {
		in, out := &in.DynamicConfigs, &out.DynamicConfigs
		*out = new(DynamicConfigsSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.HTTP != nil {
		in, out := &in.HTTP, &out.HTTP
		*out = new(HTTPSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BootstrapConfigSpec.
func (in *BootstrapConfigSpec) DeepCopy() *BootstrapConfigSpec {
	if in == nil {
		return nil
	}
	out := new(BootstrapConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Concurrency) DeepCopyInto(out *Concurrency) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Concurrency.
func (in *Concurrency) DeepCopy() *Concurrency {
	if in == nil {
		return nil
	}
	out := new(Concurrency)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerManagerConfigSpec) DeepCopyInto(out *ControllerManagerConfigSpec) {
	*out = *in
	if in.LeaderElection != nil {
		in, out := &in.LeaderElection, &out.LeaderElection
		*out = new(LeaderElectionSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerManagerConfigSpec.
func (in *ControllerManagerConfigSpec) DeepCopy() *ControllerManagerConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ControllerManagerConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronControllerConfig) DeepCopyInto(out *CronControllerConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.CronHashNames != nil {
		in, out := &in.CronHashNames, &out.CronHashNames
		*out = new(bool)
		**out = **in
	}
	if in.CronHashSecondsByDefault != nil {
		in, out := &in.CronHashSecondsByDefault, &out.CronHashSecondsByDefault
		*out = new(bool)
		**out = **in
	}
	if in.CronHashFields != nil {
		in, out := &in.CronHashFields, &out.CronHashFields
		*out = new(bool)
		**out = **in
	}
	if in.DefaultTimezone != nil {
		in, out := &in.DefaultTimezone, &out.DefaultTimezone
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronControllerConfig.
func (in *CronControllerConfig) DeepCopy() *CronControllerConfig {
	if in == nil {
		return nil
	}
	out := new(CronControllerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CronControllerConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynamicConfigsSpec) DeepCopyInto(out *DynamicConfigsSpec) {
	*out = *in
	if in.ConfigMap != nil {
		in, out := &in.ConfigMap, &out.ConfigMap
		*out = new(ObjectReference)
		**out = **in
	}
	if in.Secret != nil {
		in, out := &in.Secret, &out.Secret
		*out = new(ObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynamicConfigsSpec.
func (in *DynamicConfigsSpec) DeepCopy() *DynamicConfigsSpec {
	if in == nil {
		return nil
	}
	out := new(DynamicConfigsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExecutionControllerConcurrencySpec) DeepCopyInto(out *ExecutionControllerConcurrencySpec) {
	*out = *in
	if in.Job != nil {
		in, out := &in.Job, &out.Job
		*out = new(Concurrency)
		**out = **in
	}
	if in.JobConfig != nil {
		in, out := &in.JobConfig, &out.JobConfig
		*out = new(Concurrency)
		**out = **in
	}
	if in.JobQueue != nil {
		in, out := &in.JobQueue, &out.JobQueue
		*out = new(Concurrency)
		**out = **in
	}
	if in.Cron != nil {
		in, out := &in.Cron, &out.Cron
		*out = new(Concurrency)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExecutionControllerConcurrencySpec.
func (in *ExecutionControllerConcurrencySpec) DeepCopy() *ExecutionControllerConcurrencySpec {
	if in == nil {
		return nil
	}
	out := new(ExecutionControllerConcurrencySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExecutionControllerConfig) DeepCopyInto(out *ExecutionControllerConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.BootstrapConfigSpec.DeepCopyInto(&out.BootstrapConfigSpec)
	in.ControllerManagerConfigSpec.DeepCopyInto(&out.ControllerManagerConfigSpec)
	if in.ControllerConcurrency != nil {
		in, out := &in.ControllerConcurrency, &out.ControllerConcurrency
		*out = new(ExecutionControllerConcurrencySpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExecutionControllerConfig.
func (in *ExecutionControllerConfig) DeepCopy() *ExecutionControllerConfig {
	if in == nil {
		return nil
	}
	out := new(ExecutionControllerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExecutionControllerConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExecutionWebhookConfig) DeepCopyInto(out *ExecutionWebhookConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.BootstrapConfigSpec.DeepCopyInto(&out.BootstrapConfigSpec)
	if in.Webhooks != nil {
		in, out := &in.Webhooks, &out.Webhooks
		*out = new(WebhookServerSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExecutionWebhookConfig.
func (in *ExecutionWebhookConfig) DeepCopy() *ExecutionWebhookConfig {
	if in == nil {
		return nil
	}
	out := new(ExecutionWebhookConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExecutionWebhookConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPSpec) DeepCopyInto(out *HTTPSpec) {
	*out = *in
	if in.Metrics != nil {
		in, out := &in.Metrics, &out.Metrics
		*out = new(MetricsSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Health != nil {
		in, out := &in.Health, &out.Health
		*out = new(HealthSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPSpec.
func (in *HTTPSpec) DeepCopy() *HTTPSpec {
	if in == nil {
		return nil
	}
	out := new(HTTPSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HealthSpec) DeepCopyInto(out *HealthSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HealthSpec.
func (in *HealthSpec) DeepCopy() *HealthSpec {
	if in == nil {
		return nil
	}
	out := new(HealthSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobControllerConfig) DeepCopyInto(out *JobControllerConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobControllerConfig.
func (in *JobControllerConfig) DeepCopy() *JobControllerConfig {
	if in == nil {
		return nil
	}
	out := new(JobControllerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *JobControllerConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LeaderElectionSpec) DeepCopyInto(out *LeaderElectionSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	out.LeaseDuration = in.LeaseDuration
	out.RenewDeadline = in.RenewDeadline
	out.RetryPeriod = in.RetryPeriod
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LeaderElectionSpec.
func (in *LeaderElectionSpec) DeepCopy() *LeaderElectionSpec {
	if in == nil {
		return nil
	}
	out := new(LeaderElectionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsSpec) DeepCopyInto(out *MetricsSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsSpec.
func (in *MetricsSpec) DeepCopy() *MetricsSpec {
	if in == nil {
		return nil
	}
	out := new(MetricsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ObjectReference) DeepCopyInto(out *ObjectReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ObjectReference.
func (in *ObjectReference) DeepCopy() *ObjectReference {
	if in == nil {
		return nil
	}
	out := new(ObjectReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebhookServerSpec) DeepCopyInto(out *WebhookServerSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebhookServerSpec.
func (in *WebhookServerSpec) DeepCopy() *WebhookServerSpec {
	if in == nil {
		return nil
	}
	out := new(WebhookServerSpec)
	in.DeepCopyInto(out)
	return out
}
