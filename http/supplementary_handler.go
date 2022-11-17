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

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
)

func (s *WebconfigServer) MultipartSupplementaryHandler(w http.ResponseWriter, r *http.Request) {
	// ==== data integrity check ====
	params := mux.Vars(r)
	mac, ok := params["mac"]
	if !ok || len(mac) != 12 {
		Error(w, http.StatusNotFound, nil)
		return
	}

	// ==== processing ====
	var fields log.Fields
	if xw, ok := w.(*XResponseWriter); ok {
		fields = xw.Audit()
	} else {
		err := fmt.Errorf("MultipartConfigHandler() responsewriter cast error")
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	urlSuffix := util.GetTelemetryQueryString(r.Header, mac)
	fields["is_telemetry"] = true

	rbytes, err := s.GetProfiles(urlSuffix, fields)
	isProfileNotFound := false
	if err != nil {
		var rherr common.RemoteHttpError
		if errors.As(err, &rherr) {
			if rherr.StatusCode == http.StatusNotFound {
				isProfileNotFound = true
				rbytes = nil
			}

		}
		if !isProfileNotFound {
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}
	}

	// append profiles stored at webconfig
	xbytes, err := s.AppendProfiles(mac, rbytes)
	if err != nil {
		if errors.Is(err, common.ProfileNotFound) {
			if isProfileNotFound {
				Error(w, http.StatusNotFound, nil)
				return
			}
		} else {
			// TODO eval if any error here should be ignored/masked, for now, we give ISE/500
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}
	}

	mpart, err := util.TelemetryBytesToMultipart(xbytes)
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
	w.Header().Set("Etag", rootVersion)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}
