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
	"net/http"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
)

func (s *WebconfigServer) GetDocumentHandler(w http.ResponseWriter, r *http.Request) {
	mac, groupId, _, fields, err := Validate(w, r, false)
	if err != nil {
		var status int
		if errors.As(err, common.Http400ErrorType) {
			status = http.StatusBadRequest
		} else if errors.As(err, common.Http404ErrorType) {
			status = http.StatusNotFound
		} else if errors.As(err, common.Http500ErrorType) {
			status = http.StatusInternalServerError
		} else {
			status = http.StatusInternalServerError
		}
		Error(w, r, status, err)
		return
	}

	mdoc, err := s.GetDocument(mac, groupId, fields)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, r, http.StatusNotFound, nil)
		} else {
			LogError(w, r, err)
			Error(w, r, http.StatusInternalServerError, err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/msgpack")
	rbytes := mdoc.Bytes()
	WriteResponseBytes(w, r, rbytes, http.StatusOK, "application/msgpack")
}

func (s *WebconfigServer) PostDocumentHandler(w http.ResponseWriter, r *http.Request) {
	mac, groupId, bbytes, fields, err := Validate(w, r, true)
	if err != nil {
		var status int
		if errors.As(err, common.Http400ErrorType) {
			status = http.StatusBadRequest
		} else if errors.As(err, common.Http404ErrorType) {
			status = http.StatusNotFound
		} else if errors.As(err, common.Http500ErrorType) {
			status = http.StatusInternalServerError
		} else {
			status = http.StatusInternalServerError
		}
		Error(w, r, status, err)
		return
	}

	version := util.GetMurmur3Hash(bbytes)
	updatedTime := time.Now().UnixNano() / 1000000
	state := common.PendingDownload
	doc := common.NewDocument(
		bbytes,
		nil,
		&version,
		&state,
		&updatedTime,
	)

	err = s.SetDocument(mac, groupId, doc, fields)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, err)
		return
	}

	// update the root version
	folder, err := s.GetFolder(mac, fields)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, err)
		return
	}

	folder.SetDocument(groupId, doc)
	newRootVersion := util.RootVersion(folder.VersionMap())
	err = s.SetRootDocumentVersion(mac, newRootVersion)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, err)
		return
	}

	WriteOkResponse(w, r, nil)
}

func (s *WebconfigServer) DeleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	mac, groupId, _, fields, err := Validate(w, r, false)
	if err != nil {
		var status int
		if errors.As(err, common.Http400ErrorType) {
			status = http.StatusBadRequest
		} else if errors.As(err, common.Http404ErrorType) {
			status = http.StatusNotFound
		} else if errors.As(err, common.Http500ErrorType) {
			status = http.StatusInternalServerError
		} else {
			status = http.StatusInternalServerError
		}
		Error(w, r, status, err)
		return
	}

	err = s.DeleteDocument(mac, groupId, fields)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, r, http.StatusNotFound, nil)
		} else {
			Error(w, r, http.StatusInternalServerError, err)
		}
		return
	}

	// update the root version
	folder, err := s.GetFolder(mac, fields)
	if err != nil {
		if s.IsDbNotFound(err) {
			err := s.DeleteRootDocumentVersion(mac)
			if err != nil {
				Error(w, r, http.StatusInternalServerError, err)
			}
		} else {
			Error(w, r, http.StatusInternalServerError, err)
		}
		return
	}

	newRootVersion := util.RootVersion(folder.VersionMap())
	err = s.SetRootDocumentVersion(mac, newRootVersion)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, err)
		return
	}

	WriteOkResponse(w, r, nil)
}
