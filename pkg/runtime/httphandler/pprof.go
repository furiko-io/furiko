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
	"net/http/pprof"
	"path"

	"k8s.io/klog/v2"

	configv1alpha1 "github.com/furiko-io/furiko/apis/config/v1alpha1"
)

const (
	defaultPprofIndexPath = "/debug/pprof"
)

// ServePprof adds the pprof debug handler to the given serve mux.
func ServePprof(mux *http.ServeMux, cfg *configv1alpha1.HTTPPprofSpec) {
	// Not enabled.
	if cfg == nil {
		return
	}
	if enabled := cfg.Enabled; enabled == nil || !*enabled {
		return
	}

	indexPath := cfg.IndexPath
	if indexPath == "" {
		indexPath = defaultPprofIndexPath
	}

	// Clean the path to remove any trailing slashes.
	indexPath = path.Clean(indexPath)

	// Register all handlers.
	mux.HandleFunc(indexPath+"/", pprof.Index)
	mux.HandleFunc(indexPath+"/cmdline", pprof.Cmdline)
	mux.HandleFunc(indexPath+"/profile", pprof.Profile)
	mux.HandleFunc(indexPath+"/symbol", pprof.Symbol)
	mux.HandleFunc(indexPath+"/trace", pprof.Trace)

	klog.V(4).Infof("httphandler: added http handler for pprof")
}
