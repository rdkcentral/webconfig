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
	"fmt"
	"net/http"
	"strings"

	"github.com/rdkcentral/webconfig/common"
)

type TokenRequest struct {
	Mac       string `json:"mac"`
	Ttl       int64  `json:"ttl"`
	PartnerId string `json:"partner_id"`
}

func (s *WebconfigServer) CreateTokenHandler(w http.ResponseWriter, r *http.Request) {
	m := s.TokenManager

	xw, ok := w.(*XResponseWriter)
	if !ok {
		err := fmt.Errorf("responsewriter cast error")
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}
	bodyBytes := xw.BodyBytes()

	// Unmarshal
	tokenRequest := TokenRequest{}
	if err := json.Unmarshal(bodyBytes, &tokenRequest); err != nil {
		Error(w, http.StatusInternalServerError, err)
		return
	}

	var token string
	if len(tokenRequest.PartnerId) > 0 {
		token = m.Generate(strings.ToLower(tokenRequest.Mac), tokenRequest.Ttl, tokenRequest.PartnerId)
	} else {
		token = m.Generate(strings.ToLower(tokenRequest.Mac), tokenRequest.Ttl)
	}

	WriteOkResponse(w, token)
}
