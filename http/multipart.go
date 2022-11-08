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
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	log "github.com/sirupsen/logrus"
)

func (s *WebconfigServer) MultipartConfigHandler(w http.ResponseWriter, r *http.Request) {
	// check if this is a Supplementary service, if so, call a different handler
	if hd := r.Header.Get(common.HeaderSupplementaryService); len(hd) > 0 {
		s.MultipartSupplementaryHandler(w, r)
		return
	}

	// ==== data integrity check ====
	params := mux.Vars(r)
	mac, ok := params["mac"]
	if !ok || len(mac) != 12 {
		Error(w, http.StatusNotFound, nil)
		return
	}
	mac = strings.ToUpper(mac)
	r.Header.Set(common.HeaderDeviceId, mac)

	// ==== processing ====
	// partnerId should be in fields by middleware
	xw, ok := w.(*XpcResponseWriter)
	if !ok {
		err1 := fmt.Errorf("MultipartConfigHandler() responsewriter cast error")
		Error(w, http.StatusInternalServerError, err1)
		return
	}
	fields := xw.Audit()

	fields["cpe_mac"] = mac
	if qGroupIds, ok := r.URL.Query()["group_id"]; ok {
		fields["group_id"] = qGroupIds[0]
		r.Header.Set(common.HeaderDocName, qGroupIds[0])
	}

	dbclient := s.DatabaseClient
	uconn := s.GetUpstreamConnector()
	status, respHeader, respBytes, err := BuildWebconfigResponse(dbclient, uconn, r.Header, nil, common.RouteHttp, fields)
	if err != nil && respBytes == nil {
		respBytes = []byte(err.Error())
	}

	// REMINDER 404 use standard response
	if status == http.StatusNotFound {
		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		o := common.HttpErrorResponse{
			Status:  status,
			Message: http.StatusText(status),
			Errors:  errStr,
		}
		WriteByMarshal(w, status, o)
		return
	}

	for k := range respHeader {
		w.Header().Set(k, respHeader.Get(k))
	}

	w.WriteHeader(status)
	_, _ = w.Write(respBytes)
}

func BuildWebconfigResponse(c db.DatabaseClient, uconn *UpstreamConnector, rHeader http.Header, bbytes []byte, route string, fields log.Fields) (int, http.Header, []byte, error) {
	// do I need these 2 here?
	mac := rHeader.Get(common.HeaderDeviceId)
	respHeader := make(http.Header)

	document, postUpstream, err := db.BuildGetDocument(c, rHeader, route, fields)
	if err != nil {
		if !c.IsDbNotFound(err) {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}
		return http.StatusNotFound, respHeader, nil, common.NewError(err)
	}

	// 304
	if document.Length() == 0 {
		return http.StatusNotModified, respHeader, nil, nil
	}

	respBytes, err := document.Bytes()
	if err != nil {
		return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
	}

	if !postUpstream || uconn == nil {
		// update states to InDeployment before the final response
		if err := db.UpdateDocumentStateIndeployment(c, mac, document); err != nil {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}

		respHeader.Set("Content-type", common.MultipartContentType)
		respHeader.Set("Etag", document.RootVersion())
		return http.StatusOK, respHeader, respBytes, nil
	}

	// =============================
	// upstream handling
	// =============================
	upstreamHeaderMap := make(http.Header)
	upstreamHeaderMap.Set("Content-type", common.MultipartContentType)
	upstreamHeaderMap.Set("Etag", document.RootVersion())
	upstreamRespBytes, upstreamRespHeader, err := uconn.PostUpstream(mac, upstreamHeaderMap, respBytes, fields)
	if err != nil {
		return http.StatusInternalServerError, respHeader, respBytes, common.NewError(err)
	}

	// ==== parse the upstreamRespBytes and store them ====
	if x := upstreamRespHeader.Get(common.HeaderStoreUpstreamResponse); x == "true" {
		err := db.WriteDocumentFromUpstream(c, mac, upstreamRespHeader, upstreamRespBytes, document)
		if err != nil {
			return http.StatusInternalServerError, upstreamRespHeader, upstreamRespBytes, common.NewError(err)
		}
	}
	return http.StatusOK, upstreamRespHeader, upstreamRespBytes, nil
}
