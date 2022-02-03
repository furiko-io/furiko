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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExecutionWebhookConfig defines bootstrap configuration for execution-webhook.
type ExecutionWebhookConfig struct {
	metav1.TypeMeta `json:",inline"`

	BootstrapConfigSpec `json:",inline"`

	// Webhooks controls the webhooks server.
	// +optional
	Webhooks *WebhookServerSpec `json:"webhooks,omitempty"`
}

type WebhookServerSpec struct {
	// BindAddress is the TCP address that the controller manager should bind to for
	// serving webhook requests over HTTPS.
	//
	// Default: :9443
	// +optional
	BindAddress string `json:"bindAddress,omitempty"`

	// TLSCertFile is the path to the X.509 certificate to use for serving webhook
	// requests over HTTPS.
	TLSCertFile string `json:"tlsCertFile"`

	// TLSPrivateKeyFile is the path to the private key which corresponds to
	// TLSCertFile, to use for serving webhook requests over HTTPS.
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile"`
}

func init() {
	SchemeBuilder.Register(&ExecutionWebhookConfig{})
}
