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

	"github.com/gorilla/mux"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
)

func (s *WebconfigServer) GetRootDocumentHandler(w http.ResponseWriter, r *http.Request) {
	// ==== data integrity check ====
	params := mux.Vars(r)
	mac := params["mac"]
	if s.ValidateMacEnabled() {
		if !util.ValidateMac(mac) {
			err := *common.NewHttp400Error("invalid mac")
			Error(w, http.StatusBadRequest, common.NewError(err))
			return
		}
	}
	mac = strings.ToUpper(mac)

	// ==== read the rootdoc from db ====
	rootdoc, err := s.GetRootDocument(mac)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, http.StatusNotFound, nil)
			return
		}
		Error(w, http.StatusInternalServerError, err)
		return
	}

	WriteOkResponse(w, rootdoc)
}

func (s *WebconfigServer) PostRootDocumentHandler(w http.ResponseWriter, r *http.Request) {
	// ==== data integrity check ====
	params := mux.Vars(r)
	mac := params["mac"]
	if s.ValidateMacEnabled() {
		if !util.ValidateMac(mac) {
			err := *common.NewHttp400Error("invalid mac")
			Error(w, http.StatusBadRequest, common.NewError(err))
			return
		}
	}
	mac = strings.ToUpper(mac)

	// ==== parse the post body ====
	xw, ok := w.(*XResponseWriter)
	if !ok {
		err := *common.NewHttp500Error("responsewriter cast error")
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	bodyBytes := xw.BodyBytes()
	if len(bodyBytes) == 0 {
		err := *common.NewHttp400Error("empty body")
		Error(w, http.StatusBadRequest, common.NewError(err))
		return
	}
	var rootdoc *common.RootDocument
	err := json.Unmarshal(bodyBytes, &rootdoc)
	if err != nil {
		Error(w, http.StatusBadRequest, common.NewError(err))
		return
	}

	err = s.SetRootDocument(mac, rootdoc)
	if err != nil {
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	WriteOkResponse(w, rootdoc)
}
