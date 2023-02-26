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

	"github.com/pkg/errors"
)

// tlsServer is a wrapper around *http.Server that overrides the ListenAndServeHTTP
// implementation for TLS.
type tlsServer struct {
	*http.Server
	certFile string
	keyFile  string
}

func newTLSServer(server *http.Server, certFile, keyFile string) *tlsServer {
	return &tlsServer{
		Server:   server,
		certFile: certFile,
		keyFile:  keyFile,
	}
}

var _ ListeningServer = (*tlsServer)(nil)

func (s *tlsServer) ListenAndServe() error {
	if s.certFile == "" {
		return errors.New("certFile must be specified")
	}
	if s.keyFile == "" {
		return errors.New("keyFile must be specified")
	}
	return s.ListenAndServeTLS(s.certFile, s.keyFile)
}
