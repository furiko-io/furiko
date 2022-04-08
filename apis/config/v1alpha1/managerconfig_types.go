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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExecutionControllerConfig defines bootstrap configuration for execution-controller.
type ExecutionControllerConfig struct {
	metav1.TypeMeta `json:",inline"`

	BootstrapConfigSpec         `json:",inline"`
	ControllerManagerConfigSpec `json:",inline"`

	// ControllerConcurrency defines the concurrency factor for individual controllers.
	// +optional
	ControllerConcurrency *ExecutionControllerConcurrencySpec `json:"controllerConcurrency,omitempty"`
}

// BootstrapConfigSpec is a shared configuration spec for all controller
// managers and webhook servers.
type BootstrapConfigSpec struct {
	// DefaultResync controls the default resync duration.
	//
	// Default: 10 minutes
	// +optional
	DefaultResync metav1.Duration `json:"defaultResync,omitempty"`

	// DynamicConfigs defines how to load dynamic configs.
	// +optional
	DynamicConfigs *DynamicConfigsSpec `json:"dynamicConfigs,omitempty"`

	// HTTP controls HTTP serving.
	// +optional
	HTTP *HTTPSpec `json:"http,omitempty"`
}

// ControllerManagerConfigSpec is a shared configuration spec for all controller managers.
type ControllerManagerConfigSpec struct {
	// LeaderElection controls leader election configuration.
	// +optional
	LeaderElection *LeaderElectionSpec `json:"leaderElection,omitempty"`
}

type LeaderElectionSpec struct {
	// Enabled controls whether leader election is enabled.
	//
	// Default: false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// LeaseName controls the name used for the lease.
	// If left empty, then a default name will be used.
	//
	// +optional
	LeaseName string `json:"leaseName,omitempty"`

	// LeaseNamespace controls the namespace used for the lease.
	//
	// Default: furiko-system
	// +optional
	LeaseNamespace string `json:"leaseNamespace,omitempty"`

	// LeaseDuration is the duration that non-leader candidates will wait after
	// observing a leadership renewal until attempting to acquire leadership of a
	// led but unrenewed leader slot. This is effectively the maximum duration that
	// a leader can be stopped before it is replaced by another candidate. This is
	// only applicable if leader election is enabled.
	//
	// Default: 30s
	// +optional
	LeaseDuration metav1.Duration `json:"leaseDuration,omitempty"`

	// RenewDeadline is the interval between attempts by the acting master to renew
	// a leadership slot before it stops leading. This must be less than or equal to
	// the lease duration. This is only applicable if leader election is enabled.
	//
	// Default: 15s
	// +optional
	RenewDeadline metav1.Duration `json:"renewDeadline,omitempty"`

	// RetryPeriod is the duration the clients should wait between attempting
	// acquisition and renewal of a leadership. This is only applicable if leader
	// election is enabled.
	//
	// Default: 5s
	// +optional
	RetryPeriod metav1.Duration `json:"retryPeriod,omitempty"`
}

type DynamicConfigsSpec struct {
	// Defines how the dynamic ConfigMap is loaded.
	//
	// Defaults to:
	//  - namespace: furiko-system
	//  - name: execution-dynamic-config
	//
	// +optional
	ConfigMap *ObjectReference `json:"configMap,omitempty"`

	// Defines how the dynamic Secret is loaded. If the Secret is present, fields
	// take precedence over those defined in ConfigMap.
	//
	// Defaults to:
	//  - namespace: furiko-system
	//  - name: execution-dynamic-config
	//
	// +optional
	Secret *ObjectReference `json:"secret,omitempty"`
}

type ObjectReference struct {
	// Namespace of the object. If empty, the default namespace will be used.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the object.
	Name string `json:"name"`
}

type HTTPSpec struct {
	// BindAddress is the TCP address that the controller manager should bind to for
	// serving HTTP requests.
	//
	// Default: :8080
	// +optional
	BindAddress string `json:"bindAddress,omitempty"`

	// Metrics controls metrics serving.
	// +optional
	Metrics *MetricsSpec `json:"metrics,omitempty"`

	// Health controls health status serving.
	// +optional
	Health *HealthSpec `json:"health,omitempty"`
}

type MetricsSpec struct {
	// Enabled is whether the controller manager enables serving Prometheus metrics.
	//
	// Default: true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// MetricsPath is the path that serves Prometheus metrics.
	//
	// Default: /metrics
	// +optional
	MetricsPath string `json:"metricsPath,omitempty"`
}

type HealthSpec struct {
	// Enabled is whether the controller manager enables serving health probes.
	//
	// Default: true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// ReadinessProbePath is the path to the readiness probe.
	//
	// Default: /readyz
	// +optional
	ReadinessProbePath string `json:"readinessProbePath,omitempty"`

	// LivenessProbePath is the path to the liveness probe.
	//
	// Default: /healthz
	// +optional
	LivenessProbePath string `json:"livenessProbePath,omitempty"`
}

type ExecutionControllerConcurrencySpec struct {
	// Control the concurrency for the Job controller.
	//
	// Default: factorOfCPUs = 4
	// +optional
	Job *Concurrency `json:"job,omitempty"`

	// Control the concurrency for the JobConfig controller.
	//
	// Default: factorOfCPUs = 4
	// +optional
	JobConfig *Concurrency `json:"jobConfig,omitempty"`

	// Control the concurrency for the JobQueue controller.
	//
	// Default: factorOfCPUs = 4
	// +optional
	JobQueue *Concurrency `json:"jobQueue,omitempty"`

	// Control the concurrency for the Cron controller.
	//
	// Default: factorOfCPUs = 4
	// +optional
	Cron *Concurrency `json:"cron,omitempty"`
}

type Concurrency struct {
	// Define an absolute number of workers for the controller.
	// Takes precedence over FactorOfCPUs if it is also defined.
	//
	// +optional
	Workers uint64 `json:"workers,omitempty"`

	// Define the number of workers as a factor of the number of CPUs. This is
	// useful to scale a controller vertically that is CPU-bound with a single CPU
	// count knob, and tie the number of workers based on the total number of CPUs.
	//
	// +optional
	FactorOfCPUs uint64 `json:"factorOfCPUs,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ExecutionControllerConfig{})
}
