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

func (s *WebconfigServer) AddBaseRoutes(testOnly bool, router *mux.Router) {
	r0 := router.Path("/monitor").Subrouter()
	r0.HandleFunc("", s.MonitorHandler).Methods("HEAD", "GET")

	r1 := router.Path("/healthz").Subrouter()
	r1.HandleFunc("", s.MonitorHandler).Methods("HEAD", "GET")

	r2 := router.Path("/version").Subrouter()
	r2.HandleFunc("", s.VersionHandler).Methods("GET")

	r3 := router.Path("/config").Subrouter()
	r3.HandleFunc("", s.ServerConfigHandler).Methods("GET")

	if s.TokenApiEnabled() {
		r4 := router.Path("/api/v1/token").Subrouter()
		r4.Use(s.NoAuthMiddleware)
		r4.HandleFunc("", s.CreateTokenHandler).Methods("POST")
	}

	// TODO remove this debug handler
	r5 := router.Path("/notif").Subrouter()
	r5.HandleFunc("", s.NotificationHandler).Methods("GET")

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

	// provide read capability to check the local fw cache
	sub3 := router.Path("/api/v1/device/{mac}/supported_groups").Subrouter()
	if testOnly {
		sub3.Use(s.TestingMiddleware)
	} else {
		if s.ServerApiTokenAuthEnabled() {
			sub3.Use(s.ApiMiddleware)
		} else {
			sub3.Use(s.NoAuthMiddleware)
		}
	}
	sub3.HandleFunc("", s.GetSupportedGroupsHandler).Methods("GET")
}

func (s *WebconfigServer) GetRouter(testOnly bool) *mux.Router {
	router := mux.NewRouter()
	s.AddBaseRoutes(testOnly, router)

	// route handlers here could be overridden
	sub1 := router.Path("/api/v1/device/{mac}/document/{subdoc_id}").Subrouter()
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

	sub2 := router.Path("/api/v1/device/{mac}/poke").Subrouter()
	if testOnly {
		sub2.Use(s.TestingMiddleware)
	} else {
		if s.ServerApiTokenAuthEnabled() {
			sub2.Use(s.ApiMiddleware)
		} else {
			sub2.Use(s.NoAuthMiddleware)
		}
	}
	sub2.HandleFunc("", s.PokeHandler).Methods("POST")

	sub3 := router.Path("/api/v1/device/{mac}/rootdocument").Subrouter()
	if testOnly {
		sub3.Use(s.TestingMiddleware)
	} else {
		if s.ServerApiTokenAuthEnabled() {
			sub3.Use(s.ApiMiddleware)
		} else {
			sub3.Use(s.NoAuthMiddleware)
		}
	}
	sub3.HandleFunc("", s.GetRootDocumentHandler).Methods("GET")

	return router
}
