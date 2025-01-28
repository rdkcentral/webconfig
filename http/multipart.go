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
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
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
	if !ok {
		Error(w, http.StatusNotFound, nil)
		return
	}
	mac = strings.ToUpper(mac)
	if s.ValidateMacEnabled() {
		if !util.ValidateMac(mac) {
			err := *common.NewHttp400Error("invalid mac")
			Error(w, http.StatusBadRequest, common.NewError(err))
			return
		}
	}
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

	// enforce strict query parameters check
	err := util.ValidateQueryParams(r, s.ValidSubdocIdMap(), fields)
	if err != nil && s.QueryParamsValidationEnabled() {
		if errors.Is(err, common.ErrInvalidQueryParams) {
			Error(w, http.StatusBadRequest, nil)
			log.WithFields(fields).Error(err)
			return
		}
		Error(w, http.StatusInternalServerError, err)
		return
	}

	// handle empty schema version header
	if x := r.Header.Get(common.HeaderSchemaVersion); len(x) == 0 {
		r.Header.Set(common.HeaderSchemaVersion, "none")
	}

	status, respHeader, respBytes, err := BuildWebconfigResponse(s, r.Header, common.RouteHttp, fields)

	switch status {
	case http.StatusNotFound:
		Error(w, status, nil)
		return
	case http.StatusConflict:
		w.WriteHeader(status)
		return
	}

	for k := range respHeader {
		w.Header().Set(k, respHeader.Get(k))
	}

	if err != nil && respBytes == nil {
		Error(w, status, common.NewError(err))
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(respBytes)
}

func BuildWebconfigResponse(s *WebconfigServer, rHeader http.Header, route string, fields log.Fields) (int, http.Header, []byte, error) {
	fields["for_device"] = true
	fields["is_primary"] = true

	c := s.DatabaseClient
	uconn := s.GetUpstreamConnector()
	mac := rHeader.Get(common.HeaderDeviceId)
	respHeader := make(http.Header)
	userAgent := rHeader.Get("User-Agent")

	// factory reset handling
	ifNoneMatch := rHeader.Get(common.HeaderIfNoneMatch)
	if ifNoneMatch == "NONE" || ifNoneMatch == "NONE-REBOOT" {
		status, respHeader, rbytes, err := BuildFactoryResetResponse(s, rHeader, fields)
		if err != nil {
			return status, respHeader, rbytes, common.NewError(err)
		}
		return status, respHeader, rbytes, nil
	}

	document, oldRootDocument, newRootDocument, deviceVersionMap, postUpstream, messages, err := db.BuildGetDocument(c, rHeader, route, fields)
	if s.KafkaProducerEnabled() && s.StateCorrectionEnabled() && len(messages) > 0 {
		s.ForwardSuccessKafkaMessages(messages, fields)
	}

	// root_document locked
	if errors.Is(err, common.ErrRootDocumentLocked) {
		return http.StatusConflict, respHeader, nil, common.NewError(err)
	}

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

		// filter blockedIds
		for _, subdocId := range c.BlockedSubdocIds() {
			document.DeleteSubDocument(subdocId)
		}

		document, err = db.LoadRefSubDocuments(c, document, fields)
		if err != nil {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}
		respBytes, err := document.Bytes()
		if err != nil {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}

		// skip updating states
		if userAgent != "mget" {
			if err := db.UpdateDocumentStateIndeployment(c, mac, document, fields); err != nil {
				return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
			}
		}

		respHeader.Set(common.HeaderContentType, common.MultipartContentType)
		respHeader.Set(common.HeaderEtag, document.RootVersion())
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
		rootDocument := common.NewRootDocument(0, "", "", "", "", "", "")
		document = common.NewDocument(rootDocument)
	}

	if userAgent == "mget" {
		postUpstream = false
	}

	var respBytes []byte
	respStatus := http.StatusNotModified
	if document.Length() > 0 {

		if !postUpstream {
			document, err = db.LoadRefSubDocuments(c, document, fields)
			if err != nil {
				return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
			}
		}
		respBytes, err = document.Bytes()
		if err != nil {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}
		respStatus = http.StatusOK
	} else if len(document.RootVersion()) == 0 {
		respStatus = http.StatusNotFound
	}

	// mget ==> no upstream
	if userAgent == "mget" {
		respHeader.Set(common.HeaderContentType, common.MultipartContentType)
		respHeader.Set(common.HeaderEtag, document.RootVersion())
		return http.StatusOK, respHeader, respBytes, nil
	}

	if !postUpstream {
		// update states to InDeployment before the final response
		if err := db.UpdateDocumentStateIndeployment(c, mac, document, fields); err != nil {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}

		respHeader.Set(common.HeaderContentType, common.MultipartContentType)
		respHeader.Set(common.HeaderEtag, document.RootVersion())
		return respStatus, respHeader, respBytes, nil
	}

	// =============================
	// upstream handling
	// =============================
	upstreamHeader := rHeader.Clone()
	upstreamHeader.Set(common.HeaderContentType, common.MultipartContentType)
	upstreamHeader.Set(common.HeaderEtag, document.RootVersion())
	if itf, ok := fields["audit_id"]; ok {
		auditId := itf.(string)
		if len(auditId) > 0 {
			upstreamHeader.Set(common.HeaderAuditid, auditId)
		}
	}
	if x := rHeader.Get(common.HeaderTransactionId); len(x) > 0 {
		upstreamHeader.Set(common.HeaderTransactionId, x)
	}

	if s.TokenManager != nil {
		token := rHeader.Get("Authorization")
		if len(token) > 0 {
			upstreamHeader.Set("Authorization", token)
		} else {
			token = s.Generate(mac, 86400)
			upstreamHeader.Set("Authorization", "Bearer "+token)
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
			return rherr.StatusCode, respHeader, nil, common.NewError(err)
		}
		return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
	}

	// ==== parse the upstreamRespBytes and store them ====
	finalMparts, err := util.ParseMultipartAsList(upstreamRespHeader, upstreamRespBytes)
	if err != nil {
		return http.StatusInternalServerError, respHeader, respBytes, common.NewError(err)
	}
	upstreamRespEtag := upstreamRespHeader.Get(common.HeaderEtag)

	// filter by versionMap and filter by blockedIds
	finalRootDocument := common.NewRootDocument(0, "", "", "", "", upstreamRespEtag, "")
	finalDocument := common.NewDocument(finalRootDocument)
	finalDocument.SetSubDocuments(finalMparts)

	// there are special use cases when we do not want to update subdocuments
	if upstreamRespHeader.Get(common.HeaderUpstreamResponse) != common.SkipDbUpdate {
		// update states based on the final document
		err = db.WriteDocumentFromUpstream(c, mac, upstreamRespEtag, finalDocument, document, false, deviceVersionMap, fields)
		if err != nil {
			return http.StatusInternalServerError, upstreamRespHeader, upstreamRespBytes, common.NewError(err)
		}
	}

	finalFilteredDocument := finalDocument.FilterForGet(deviceVersionMap)
	for _, subdocId := range c.BlockedSubdocIds() {
		finalFilteredDocument.DeleteSubDocument(subdocId)
	}

	// 304
	if finalFilteredDocument.Length() == 0 {
		return http.StatusNotModified, upstreamRespHeader, nil, nil
	}

	finalFilteredDocument, err = db.LoadRefSubDocuments(c, finalFilteredDocument, fields)
	if err != nil {
		return http.StatusInternalServerError, upstreamRespHeader, nil, common.NewError(err)
	}
	finalFilteredBytes, err := finalFilteredDocument.Bytes()
	if err != nil {
		return http.StatusInternalServerError, upstreamRespHeader, finalFilteredBytes, common.NewError(err)
	}

	return http.StatusOK, upstreamRespHeader, finalFilteredBytes, nil
}

func BuildFactoryResetResponse(s *WebconfigServer, rHeader http.Header, fields log.Fields) (int, http.Header, []byte, error) {
	c := s.DatabaseClient
	uconn := s.GetUpstreamConnector()
	mac := rHeader.Get(common.HeaderDeviceId)
	respHeader := make(http.Header)

	fieldsDict := make(util.Dict)
	fieldsDict.Update(fields)
	partnerId := rHeader.Get(common.HeaderPartnerID)
	if len(partnerId) == 0 {
		partnerId = fieldsDict.GetString("partner")
	}

	rootDocument, err := db.PreprocessRootDocument(c, rHeader, mac, partnerId, fields)
	if err != nil {
		return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
	}

	document, err := c.GetDocument(mac, fields)
	if err != nil {
		if s.IsDbNotFound(err) {
			return http.StatusNotFound, respHeader, nil, nil
		} else {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}
	}
	if document == nil {
		document = common.NewDocument(rootDocument)
	} else {
		document.SetRootDocument(rootDocument)
	}

	oldDocBytes, err := document.Bytes()
	if err != nil {
		return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
	}

	if uconn == nil {
		err := c.DeleteDocument(mac)
		if err != nil {
			return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
		}
		return http.StatusNotFound, respHeader, nil, nil
	}

	// =============================
	// upstream handling
	// =============================
	upstreamHeader := rHeader.Clone()
	upstreamHeader.Set(common.HeaderContentType, common.MultipartContentType)
	upstreamHeader.Set(common.HeaderEtag, document.RootVersion())
	upstreamHeader.Set(common.HeaderUpstreamNewPartnerId, partnerId)

	if itf, ok := fields["audit_id"]; ok {
		auditId := itf.(string)
		if len(auditId) > 0 {
			upstreamHeader.Set(common.HeaderAuditid, auditId)
		}
	}

	if s.TokenManager != nil {
		token := rHeader.Get("Authorization")
		if len(token) > 0 {
			upstreamHeader.Set("Authorization", token)
		} else {
			token = s.Generate(mac, 86400)
			upstreamHeader.Set("Authorization", "Bearer "+token)
		}
	}

	// call /upstream to handle factory reset
	upstreamRespBytes, upstreamRespHeader, err := s.PostUpstream(mac, upstreamHeader, oldDocBytes, fields)
	if err != nil {
		var rherr common.RemoteHttpError
		if errors.As(err, &rherr) {
			return rherr.StatusCode, respHeader, nil, common.NewError(err)
		}
		return http.StatusInternalServerError, respHeader, nil, common.NewError(err)
	}

	// ==== parse the upstreamRespBytes and store them ====
	finalMparts, err := util.ParseMultipartAsList(upstreamRespHeader, upstreamRespBytes)
	if err != nil {
		return http.StatusInternalServerError, respHeader, oldDocBytes, common.NewError(err)
	}
	upstreamRespEtag := upstreamRespHeader.Get(common.HeaderEtag)

	// filter by versionMap and filter by blockedIds
	finalRootDocument := common.NewRootDocument(0, "", "", "", "", upstreamRespEtag, "")
	finalDocument := common.NewDocument(finalRootDocument)
	finalDocument.SetSubDocuments(finalMparts)
	for _, subdocId := range c.BlockedSubdocIds() {
		finalDocument.DeleteSubDocument(subdocId)
	}

	// there are special use cases when we do not want to update subdocuments
	if upstreamRespHeader.Get(common.HeaderUpstreamResponse) != common.SkipDbUpdate {
		// update states based on the final document
		err = db.WriteDocumentFromUpstream(c, mac, upstreamRespEtag, finalDocument, document, true, nil, fields)
		if err != nil {
			return http.StatusInternalServerError, upstreamRespHeader, upstreamRespBytes, common.NewError(err)
		}
	}

	if finalDocument.Length() == 0 {
		return http.StatusNotFound, upstreamRespHeader, nil, nil
	}

	finalDocument, err = db.LoadRefSubDocuments(c, finalDocument, fields)
	if err != nil {
		return http.StatusInternalServerError, upstreamRespHeader, nil, common.NewError(err)
	}
	finalBytes, err := finalDocument.Bytes()
	if err != nil {
		return http.StatusInternalServerError, upstreamRespHeader, finalBytes, common.NewError(err)
	}

	return http.StatusOK, upstreamRespHeader, finalBytes, nil
}
