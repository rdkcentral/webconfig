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
	"encoding/json"
	"net/http"
	"strings"
)

type PostTokenBody struct {
	Mac string `json:"mac"`
	Ttl int64  `json:"ttl"`
}

func (s *WebconfigServer) CreateTokenHandler(w http.ResponseWriter, r *http.Request) {
	m := s.TokenManager
	var body string
	if xw, ok := w.(*XpcResponseWriter); ok {
		body = xw.Body()
	} else {
		Error(w, r, http.StatusBadRequest, nil)
		return
	}

	// Unmarshal
	msg := PostTokenBody{}
	if err := json.Unmarshal([]byte(body), &msg); err != nil {
		Error(w, r, http.StatusBadRequest, nil)
		return
	}

	token := m.Generate(strings.ToLower(msg.Mac), msg.Ttl)
	WriteOkResponse(w, r, token)
}
