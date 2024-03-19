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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func (s *WebconfigServer) MultipartSupplementaryHandler(w http.ResponseWriter, r *http.Request) {
	// ==== data integrity check ====
	params := mux.Vars(r)
	mac, ok := params["mac"]
	if !ok {
		Error(w, http.StatusNotFound, nil)
		return
	}
	mac = strings.ToUpper(mac)

	// ==== processing ====
	var fields log.Fields
	if xw, ok := w.(*XResponseWriter); ok {
		fields = xw.Audit()
	} else {
		err := fmt.Errorf("MultipartConfigHandler() responsewriter cast error")
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	// append the extra query_params if any
	var queryParams string
	if s.SupplementaryAppendingEnabled() {
		rootdoc, err := s.GetRootDocument(mac)
		if err != nil {
			if !s.IsDbNotFound(err) {
				Error(w, http.StatusInternalServerError, common.NewError(err))
				return
			}
		}
		if rootdoc != nil {
			queryParams = rootdoc.QueryParams
		}
	}

	// partner handling
	partnerId := r.Header.Get(common.HeaderPartnerID)
	if err := s.ValidatePartner(partnerId); err != nil {
		partnerId = ""
	}

	urlSuffix := util.GetTelemetryQueryString(r.Header, mac, queryParams, partnerId)
	fields["is_telemetry"] = true

	rbytes, resHeader, err := s.GetProfiles(urlSuffix, fields)
	if err != nil {
		var rherr common.RemoteHttpError
		if errors.As(err, &rherr) {
			if rherr.StatusCode == http.StatusNotFound {
				Error(w, http.StatusNotFound, nil)
				return
			}

		}
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	mpart, err := util.TelemetryBytesToMultipart(rbytes)
	if err != nil {
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}
	mparts := []common.Multipart{
		mpart,
	}

	respBytes, err := common.WriteMultipartBytes(mparts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	rootVersion := util.GetRandomRootVersion()
	w.Header().Set("Content-type", common.MultipartContentType)
	w.Header().Set(common.HeaderEtag, rootVersion)

	// help with unit tests
	if x := resHeader.Get(common.HeaderReqUrl); len(x) > 0 {
		w.Header().Set(common.HeaderReqUrl, x)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}
