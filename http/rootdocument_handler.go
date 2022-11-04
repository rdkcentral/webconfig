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
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func (s *WebconfigServer) GetRootDocumentHandler(w http.ResponseWriter, r *http.Request) {
	// xw, ok := w.(*XpcResponseWriter)
	// if !ok {
	// 	err := fmt.Errorf("responsewriter cast error")
	// 	Error(w, http.StatusInternalServerError, common.NewError(err))
	// 	return
	// }

	// ==== data integrity check ====
	params := mux.Vars(r)
	mac := params["mac"]
	if len(mac) != 12 {
		Error(w, http.StatusNotFound, nil)
		return
	}
	// in xpcdb, all data are stored with uppercased cpemac
	mac = strings.ToUpper(mac)

	// ==== read the rootdoc from db ====
	rootdoc, err := s.GetRootDocument(mac)
	if err != nil {
		if !s.IsDbNotFound(err) {
			Error(w, http.StatusNotFound, nil)
			return
		}
		Error(w, http.StatusInternalServerError, err)
		return
	}

	WriteOkResponse(w, rootdoc)
}
