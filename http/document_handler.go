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
	"strings"
	"time"

	"github.com/gorilla/mux"
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
	if subdoc.Expiry() != nil {
		w.Header().Set(common.HeaderSubdocumentExpiry, strconv.Itoa(*subdoc.Expiry()))
	}
}

func (s *WebconfigServer) GetSubDocumentHandler(w http.ResponseWriter, r *http.Request) {
	mac, subdocId, _, _, err := s.Validate(w, r, false)
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

	subdoc, err := s.GetSubDocument(mac, subdocId)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, http.StatusNotFound, nil)
		} else {
			LogError(w, err)
			Error(w, http.StatusInternalServerError, common.NewError(err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/msgpack")
	writeStateHeaders(w, subdoc)
	w.WriteHeader(http.StatusOK)
	w.Write(subdoc.Payload())
}

func (s *WebconfigServer) PostSubDocumentHandler(w http.ResponseWriter, r *http.Request) {
	mac, subdocId, bbytes, fields, err := s.Validate(w, r, true)
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

	deviceIds := []string{
		mac,
	}
	if x, ok := r.URL.Query()["device_id"]; ok {
		elements := strings.Split(x[0], ",")
		if len(elements) > 0 {
			deviceIds = append(deviceIds, elements...)
		}
	}

	// handle version header
	version := r.Header.Get(common.HeaderSubdocumentVersion)
	if len(version) == 0 {
		version = util.GetMurmur3Hash(bbytes)
	}
	state := common.PendingDownload
	statePtr := &state
	if x := r.Header.Get(common.HeaderSubdocumentState); len(x) > 0 {
		if x == "null" {
			statePtr = nil
		} else {
			if i, err := strconv.Atoi(x); err == nil {
				statePtr = &i
			}
		}
	}

	updatedTime := int(time.Now().UnixNano() / 1000000)
	subdoc := common.NewSubDocument(bbytes, &version, statePtr, &updatedTime, nil, nil)

	// handle expiry header
	expiryTmsStr := r.Header.Get(common.HeaderSubdocumentExpiry)
	if len(expiryTmsStr) > 0 {
		expiryTms, err := strconv.Atoi(expiryTmsStr)
		if err != nil {
			Error(w, http.StatusBadRequest, common.NewError(err))
			return
		}
		subdoc.SetExpiry(&expiryTms)
	}

	oldState := 0
	if x := r.Header.Get(common.HeaderSubdocumentOldState); len(x) > 0 {
		if i, err := strconv.Atoi(x); err == nil {
			oldState = i
		}
	}

	metricsAgent := r.Header.Get(common.HeaderMetricsAgent)

	for _, deviceId := range deviceIds {
		err = s.SetSubDocument(deviceId, subdocId, subdoc, oldState, metricsAgent, fields)
		if err != nil {
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}

		// update the root version
		doc, err := s.GetDocument(deviceId, true)
		if err != nil {
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}

		doc.SetSubDocument(subdocId, subdoc)
		newRootVersion := db.HashRootVersion(doc.VersionMap())
		err = s.SetRootDocumentVersion(deviceId, newRootVersion)
		if err != nil {
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}
	}

	WriteOkResponse(w, nil)
}

func (s *WebconfigServer) DeleteSubDocumentHandler(w http.ResponseWriter, r *http.Request) {
	mac, subdocId, _, _, err := s.Validate(w, r, false)
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

	err = s.DeleteSubDocument(mac, subdocId)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, http.StatusNotFound, nil)
		} else {
			Error(w, http.StatusInternalServerError, common.NewError(err))
		}
		return
	}

	// update the root version
	doc, err := s.GetDocument(mac)
	if err != nil {
		if s.IsDbNotFound(err) {
			err := s.DeleteRootDocumentVersion(mac)
			if err != nil {
				Error(w, http.StatusInternalServerError, common.NewError(err))
			}
		} else {
			Error(w, http.StatusInternalServerError, common.NewError(err))
		}
		return
	}

	newRootVersion := db.HashRootVersion(doc.VersionMap())
	err = s.SetRootDocumentVersion(mac, newRootVersion)
	if err != nil {
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	WriteOkResponse(w, nil)
}

func (s *WebconfigServer) DeleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	mac := params["mac"]
	mac = strings.ToUpper(mac)
	if !util.ValidateMac(mac) {
		err := common.Http400Error{
			Message: "invalid mac",
		}
		Error(w, http.StatusBadRequest, common.NewError(err))
		return
	}

	err := s.DeleteDocument(mac)
	if err != nil {
		if s.IsDbNotFound(err) {
			Error(w, http.StatusNotFound, nil)
		} else {
			Error(w, http.StatusInternalServerError, common.NewError(err))
		}
		return
	}

	// update the root version
	doc, err := s.GetDocument(mac)
	if err != nil {
		if s.IsDbNotFound(err) {
			err := s.DeleteRootDocumentVersion(mac)
			if err != nil {
				Error(w, http.StatusInternalServerError, common.NewError(err))
			}
		} else {
			Error(w, http.StatusInternalServerError, common.NewError(err))
		}
		return
	}

	newRootVersion := db.HashRootVersion(doc.VersionMap())
	err = s.SetRootDocumentVersion(mac, newRootVersion)
	if err != nil {
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	WriteOkResponse(w, nil)
}
