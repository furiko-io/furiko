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
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
)

// Server is a thin wrapper around HTTP servers.
type Server struct {
	addr   string
	mux    *http.ServeMux
	server ListeningServer
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &Server{
		mux:    mux,
		addr:   addr,
		server: server,
	}
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (s *Server) Handle(path string, handler http.Handler) {
	s.mux.Handle(path, handler)
}

// RegisterCommonHandlers registers common handlers from the given HTTPSpec.
func (s *Server) RegisterCommonHandlers(
	cfg *configv1alpha1.HTTPSpec,
	registry prometheus.Gatherer,
	healthHandler HealthHandler,
	debugHandler DebugHandler,
) {
	if cfg == nil {
		cfg = defaultHTTPConfig
	}
	ServeMetrics(s.mux, cfg.Metrics, registry)
	ServeHealth(s.mux, cfg.Health, healthHandler)
	ServeDebug(s.mux, cfg.Debug, debugHandler)
	ServePprof(s.mux, cfg.Pprof)
}

// ListenAndServe will listen and serve incoming HTTP connections until the context is closed.
func (s *Server) ListenAndServe(ctx context.Context) error {
	return listenAndServe(ctx, s.addr, s.server)
}
