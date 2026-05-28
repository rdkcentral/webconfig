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

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
)

func (s *WebconfigServer) GetRefSubDocumentHandler(w http.ResponseWriter, r *http.Request) {
	refId, _, _, err := s.ValidateRefData(w, r, false)
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
		Error(w, status, common.NewError(err))
		return
	}

	refsubdoc, err := s.GetRefSubDocument(refId)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, http.StatusNotFound, nil)
		} else {
			LogError(w, err)
			Error(w, http.StatusInternalServerError, common.NewError(err))
		}
		return
	}

	w.Header().Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	if refsubdoc.Version() != nil {
		w.Header().Set(common.HeaderRefSubdocumentVersion, *refsubdoc.Version())
	}
	w.WriteHeader(http.StatusOK)
	w.Write(refsubdoc.Payload())
}

func (s *WebconfigServer) PostRefSubDocumentHandler(w http.ResponseWriter, r *http.Request) {
	refId, bbytes, _, err := s.ValidateRefData(w, r, true)
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
		Error(w, status, common.NewError(err))
		return
	}

	// handle version header
	version := r.Header.Get(common.HeaderSubdocumentVersion)
	if len(version) == 0 {
		version = util.GetMurmur3Hash(bbytes)
	}

	refsubdoc := common.NewRefSubDocument(bbytes, &version)

	err = s.SetRefSubDocument(refId, refsubdoc)
	if err != nil {
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	WriteOkResponse(w, nil)
}

func (s *WebconfigServer) DeleteRefSubDocumentHandler(w http.ResponseWriter, r *http.Request) {
	refId, _, _, err := s.ValidateRefData(w, r, false)
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
		Error(w, status, common.NewError(err))
		return
	}

	err = s.DeleteRefSubDocument(refId)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, http.StatusNotFound, nil)
		} else {
			Error(w, http.StatusInternalServerError, common.NewError(err))
		}
		return
	}
	WriteOkResponse(w, nil)
}
