/**
* Copyright 2021 Comcast Cable Communications Management, LLC
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
* SPDX-License-Identifier: Apache-2.0
*/
package http

import (
	"github.com/gorilla/mux"
)

func (s *WebconfigServer) GetBaseRouter(testOnly bool) *mux.Router {
	// setup router
	router := mux.NewRouter()
	r0 := router.Path("/monitor").Subrouter()
	r0.HandleFunc("", s.MonitorHandler).Methods("HEAD", "GET")

	r1 := router.Path("/healthz").Subrouter()
	r1.HandleFunc("", s.MonitorHandler).Methods("HEAD", "GET")

	r2 := router.Path("/api/v1/version").Subrouter()
	r2.HandleFunc("", s.VersionHandler).Methods("GET")

	r3 := router.Path("/api/v1/config").Subrouter()
	r3.HandleFunc("", s.ServerConfigHandler).Methods("GET")

	if s.TokenApiEnabled() {
		s1 := router.Path("/api/v1/token").Subrouter()
		s1.Use(s.NoAuthMiddleware)
		s1.HandleFunc("", s.CreateTokenHandler).Methods("POST")
	}

	// msgpack multipart
	sub2 := router.Path("/api/v1/device/{mac}/config").Subrouter()
	if testOnly {
		sub2.Use(s.TestingMiddleware)
	} else {
		if s.DeviceApiTokenAuthEnabled() {
			sub2.Use(s.CpeMiddleware)
		} else {
			sub2.Use(s.NoAuthMiddleware)
		}
	}
	sub2.HandleFunc("", s.MultipartConfigHandler).Methods("GET")

	// new poke for root group
	sub3 := router.Path("/api/v1/device/{mac}/poke").Subrouter()
	if testOnly {
		sub3.Use(s.TestingMiddleware)
	} else {
		if s.ServerApiTokenAuthEnabled() {
			sub3.Use(s.ApiMiddleware)
		} else {
			sub3.Use(s.NoAuthMiddleware)
		}
	}
	sub3.HandleFunc("", s.PokeHandler).Methods("POST")

	// provide read capability to check the local fw cache
	sub4 := router.Path("/api/v1/device/{mac}/supported_groups").Subrouter()
	if testOnly {
		sub4.Use(s.TestingMiddleware)
	} else {
		if s.ServerApiTokenAuthEnabled() {
			sub4.Use(s.ApiMiddleware)
		} else {
			sub4.Use(s.NoAuthMiddleware)
		}
	}
	sub4.HandleFunc("", s.GetSupportedGroupsHandler).Methods("GET")

	return router
}

func (s *WebconfigServer) GetRouter(testOnly bool) *mux.Router {
	router := s.GetBaseRouter(testOnly)

	sub1 := router.Path("/api/v1/device/{mac}/document").Subrouter()
	if testOnly {
		sub1.Use(s.TestingMiddleware)
	} else {
		if s.ServerApiTokenAuthEnabled() {
			sub1.Use(s.ApiMiddleware)
		} else {
			sub1.Use(s.NoAuthMiddleware)
		}
	}
	sub1.HandleFunc("", s.GetDocumentHandler).Methods("GET")
	sub1.HandleFunc("", s.PostDocumentHandler).Methods("POST")
	sub1.HandleFunc("", s.DeleteDocumentHandler).Methods("DELETE")

	return router
}
