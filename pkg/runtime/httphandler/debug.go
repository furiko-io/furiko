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
	"net/http"
	"path"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
	"github.com/furiko-io/furiko/pkg/runtime/controllercontext"
)

const (
	defaultDebugBasePath = "/debug"
)

// DebugHandler is a delegate to fetch debug information.
type DebugHandler interface {
	GetBootstrapConfig() runtime.Object
	GetDynamicConfig() (map[configv1alpha1.ConfigName]runtime.Object, error)
}

type defaultDebugHandler struct {
	bootstrapConfig runtime.Object
	configsGetter   controllercontext.Configs
}

var _ DebugHandler = (*defaultDebugHandler)(nil)

func (h *defaultDebugHandler) GetBootstrapConfig() runtime.Object {
	return h.bootstrapConfig
}

func (h *defaultDebugHandler) GetDynamicConfig() (map[configv1alpha1.ConfigName]runtime.Object, error) {
	return h.configsGetter.AllConfigs()
}

// ServeDebug adds debug handlers to the given serve mux.
func ServeDebug(mux *http.ServeMux, cfg *configv1alpha1.HTTPDebugSpec, handler DebugHandler) {
	if cfg == nil {
		return
	}
	if enabled := cfg.Enabled; enabled == nil || !*enabled {
		return
	}

	basePath := cfg.BasePath
	if basePath == "" {
		basePath = defaultDebugBasePath
	}

	// Clean the path to remove any trailing slashes.
	basePath = path.Clean(basePath)

	server := newDebugServer(handler)
	mux.HandleFunc(basePath+"/config/bootstrap", server.HandleBootstrapConfig())
	mux.HandleFunc(basePath+"/config/dynamic", server.HandleDynamicConfig())

	klog.V(4).Infof("httphandler: added http handler for health probes")
}

type debugServer struct {
	handler DebugHandler
}

func newDebugServer(handler DebugHandler) *debugServer {
	return &debugServer{
		handler: handler,
	}
}

func (s *debugServer) HandleBootstrapConfig() http.HandlerFunc {
	return s.makeJSONHandler(s.handler.GetBootstrapConfig())
}

func (s *debugServer) HandleDynamicConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		configs, err := s.handler.GetDynamicConfig()
		if err != nil {
			s.makeErrorHandler(err).ServeHTTP(w, r)
			return
		}
		s.makeJSONHandler(configs).ServeHTTP(w, r)
	}
}

func (s *debugServer) makeJSONHandler(data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := json.Marshal(data)
		if err != nil {
			s.makeErrorHandler(err).ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func (s *debugServer) makeErrorHandler(err error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := json.Marshal(map[string]interface{}{
			"error": err.Error(),
		})
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(body)
	}
}
