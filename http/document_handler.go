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
	"strconv"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
)

// TODO
// 1. group_id related validation
// 2. mac validation
// 3. msgpack body validation

func writeStateHeaders(w http.ResponseWriter, subdoc *common.SubDocument) {
	if subdoc.Version() != nil {
		w.Header().Set(common.HeaderSubdocumentVersion, *subdoc.Version())
	}
	if subdoc.State() != nil {
		w.Header().Set(common.HeaderSubdocumentState, strconv.Itoa(*subdoc.State()))
	}
	if subdoc.UpdatedTime() != nil {
		w.Header().Set(common.HeaderSubdocumentUpdatedTime, strconv.Itoa(*subdoc.UpdatedTime()))
	}

	if subdoc.ErrorCode() != nil {
		w.Header().Set(common.HeaderSubdocumentErrorCode, strconv.Itoa(*subdoc.ErrorCode()))
	}
	if subdoc.ErrorDetails() != nil {
		w.Header().Set(common.HeaderSubdocumentErrorDetails, *subdoc.ErrorDetails())
	}
}

func (s *WebconfigServer) GetDocumentHandler(w http.ResponseWriter, r *http.Request) {
	mac, subdocId, _, _, err := Validate(w, r, false)
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
		Error(w, status, err)
		return
	}

	subdoc, err := s.GetSubDocument(mac, subdocId)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, http.StatusNotFound, nil)
		} else {
			LogError(w, err)
			Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/msgpack")
	writeStateHeaders(w, subdoc)
	w.WriteHeader(http.StatusOK)
	w.Write(subdoc.Payload())
}

func (s *WebconfigServer) PostDocumentHandler(w http.ResponseWriter, r *http.Request) {
	mac, subdocId, bbytes, _, err := Validate(w, r, true)
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
		Error(w, status, err)
		return
	}

	version := util.GetMurmur3Hash(bbytes)
	updatedTime := int(time.Now().UnixNano() / 1000000)
	state := common.PendingDownload
	subdoc := common.NewSubDocument(bbytes, &version, &state, &updatedTime, nil, nil)

	err = s.SetSubDocument(mac, subdocId, subdoc)
	if err != nil {
		Error(w, http.StatusInternalServerError, err)
		return
	}

	// update the root version
	doc, err := s.GetDocument(mac)
	if err != nil {
		Error(w, http.StatusInternalServerError, err)
		return
	}

	doc.SetSubDocument(subdocId, subdoc)
	newRootVersion := db.HashRootVersion(doc.VersionMap())
	err = s.SetRootDocumentVersion(mac, newRootVersion)
	if err != nil {
		Error(w, http.StatusInternalServerError, err)
		return
	}

	WriteOkResponse(w, nil)
}

func (s *WebconfigServer) DeleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	mac, subdocId, _, _, err := Validate(w, r, false)
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
		Error(w, status, err)
		return
	}

	err = s.DeleteSubDocument(mac, subdocId)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, http.StatusNotFound, nil)
		} else {
			Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	// update the root version
	doc, err := s.GetDocument(mac)
	if err != nil {
		if s.IsDbNotFound(err) {
			err := s.DeleteRootDocumentVersion(mac)
			if err != nil {
				Error(w, http.StatusInternalServerError, err)
			}
		} else {
			Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	newRootVersion := db.HashRootVersion(doc.VersionMap())
	err = s.SetRootDocumentVersion(mac, newRootVersion)
	if err != nil {
		Error(w, http.StatusInternalServerError, err)
		return
	}

	WriteOkResponse(w, nil)
}
