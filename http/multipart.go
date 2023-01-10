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
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
)

var (
	upstreamHeaders = []string{
		"X-System-Firmware-Version",
		"X-System-Model-Name",
		"X-System-Schema-Version",
		"X-System-Supported-Docs",
		"X-System-Product-Class",
		"Transaction-Id",
	}
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
	xw, ok := w.(*XResponseWriter)
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

	status, respHeader, respBytes, err := BuildWebconfigResponse(s, r.Header, nil, common.RouteHttp, fields)
	if err != nil && respBytes == nil {
		respBytes = []byte(err.Error())
	}

	// REMINDER 404 use standard response
	if status == http.StatusNotFound {
		Error(w, http.StatusNotFound, nil)
		return
	}

	for k := range respHeader {
		w.Header().Set(k, respHeader.Get(k))
	}

	w.WriteHeader(status)
	_, _ = w.Write(respBytes)
}

func BuildWebconfigResponse(s *WebconfigServer, rHeader http.Header, bbytes []byte, route string, fields log.Fields) (int, http.Header, []byte, error) {
	c := s.DatabaseClient
	uconn := s.GetUpstreamConnector()
	mac := rHeader.Get(common.HeaderDeviceId)
	respHeader := make(http.Header)

	document, oldRootDocument, newRootDocument, postUpstream, err := db.BuildGetDocument(c, rHeader, route, fields)
	if uconn == nil {
		if err != nil {
			if !s.IsDbNotFound(err) {
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

		// skip updating states
		userAgent := rHeader.Get("User-Agent")
		if userAgent != "mget" {
			if err := db.UpdateDocumentStateIndeployment(c, mac, document); err != nil {
				return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
			}
		}

		respHeader.Set("Content-type", common.MultipartContentType)
		respHeader.Set("Etag", document.RootVersion())
		return http.StatusOK, respHeader, respBytes, nil
	}

	if err != nil {
		if !s.IsDbNotFound(err) {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}
		// 404
		if !postUpstream {
			postUpstream = true
		}
	}
	if document == nil {
		rootDocument := common.NewRootDocument(0, "", "", "", "", "")
		document = common.NewDocument(rootDocument)
	}

	var respBytes []byte
	if document.Length() > 0 {
		respBytes, err = document.Bytes()
		if err != nil {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}
	}

	if !postUpstream {
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
	upstreamHeader := make(http.Header)
	upstreamHeader.Set("Content-type", common.MultipartContentType)
	upstreamHeader.Set("Etag", document.RootVersion())

	if s.TokenManager != nil {
		token := rHeader.Get("Authorization")
		if len(token) > 0 {
			upstreamHeader.Set("Authorization", token)
		} else {
			token = s.Generate(mac, 86400)
			rHeader.Set("Authorization", "Bearer "+token)
		}
	}

	// add old/new header/metadata in the upstream header
	if newRootDocument != nil {
		upstreamHeader.Set(common.HeaderUpstreamNewBitmap, strconv.Itoa(newRootDocument.Bitmap))
		upstreamHeader.Set(common.HeaderUpstreamNewFirmwareVersion, newRootDocument.FirmwareVersion)
		upstreamHeader.Set(common.HeaderUpstreamNewModelName, newRootDocument.ModelName)
		upstreamHeader.Set(common.HeaderUpstreamNewPartnerId, newRootDocument.PartnerId)
		upstreamHeader.Set(common.HeaderUpstreamNewSchemaVersion, newRootDocument.SchemaVersion)
	}

	if oldRootDocument != nil {
		upstreamHeader.Set(common.HeaderUpstreamOldBitmap, strconv.Itoa(oldRootDocument.Bitmap))
		upstreamHeader.Set(common.HeaderUpstreamOldFirmwareVersion, oldRootDocument.FirmwareVersion)
		upstreamHeader.Set(common.HeaderUpstreamOldModelName, oldRootDocument.ModelName)
		upstreamHeader.Set(common.HeaderUpstreamOldPartnerId, oldRootDocument.PartnerId)
		upstreamHeader.Set(common.HeaderUpstreamOldSchemaVersion, oldRootDocument.SchemaVersion)
	}

	upstreamRespBytes, upstreamRespHeader, err := s.PostUpstream(mac, upstreamHeader, respBytes, fields)
	if err != nil {
		var rherr common.RemoteHttpError
		if errors.As(err, &rherr) {
			return rherr.StatusCode, respHeader, respBytes, common.NewError(err)
		}
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
