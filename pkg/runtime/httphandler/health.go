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

package httphandler

import (
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllermanager"
)

const (
	defaultReadinessProbePath = "/readyz"
	defaultLivenessProbePath  = "/healthz"
)

// HealthHandler is a delegate to get health information.
type HealthHandler interface {
	GetReadiness() error
	GetHealth() []controllermanager.HealthStatus
}

// ServeHealth adds health probe handlers to the given serve mux.
func ServeHealth(mux *http.ServeMux, cfg *configv1alpha1.HTTPHealthSpec, handler HealthHandler) {
	// Not enabled.
	if cfg == nil {
		return
	}
	if enabled := cfg.Enabled; enabled == nil || !*enabled {
		return
	}

	readinessProbePath := cfg.ReadinessProbePath
	if readinessProbePath == "" {
		readinessProbePath = defaultReadinessProbePath
	}
	livenessProbePath := cfg.LivenessProbePath
	if livenessProbePath == "" {
		livenessProbePath = defaultLivenessProbePath
	}

	server := newHealthServer(handler)
	mux.HandleFunc(readinessProbePath, server.HandleReadinessProbes())
	mux.HandleFunc(livenessProbePath, server.HandleLivenessProbes())

	klog.V(4).Infof("httphandler: added http handler for health probes")
}

type healthServer struct {
	handler HealthHandler
}

func newHealthServer(handler HealthHandler) *healthServer {
	return &healthServer{
		handler: handler,
	}
}

func (s *healthServer) HandleReadinessProbes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusCode := http.StatusOK
		msg := "ok"
		if err := s.handler.GetReadiness(); err != nil {
			statusCode = http.StatusServiceUnavailable
			msg = fmt.Sprintf("controller manager is not ready: %v", err)
		}
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(msg))
	}
}

func (s *healthServer) HandleLivenessProbes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthy := true
		statusCode := http.StatusAccepted

		ctrlHealth := s.handler.GetHealth()
		for _, workerHealth := range ctrlHealth {
			if !workerHealth.Healthy {
				statusCode = http.StatusInternalServerError
				healthy = false
			}
		}

		w.WriteHeader(statusCode)
		body := []byte("not healthy")
		if data, err := json.Marshal(map[string]interface{}{
			"healthy": healthy,
			"status":  ctrlHealth,
		}); err == nil {
			body = data
		}
		_, _ = w.Write(body)
	}
}
