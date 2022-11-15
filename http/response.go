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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rdkcentral/webconfig/common"
)

const (
	OkResponseTemplate              = `{"status":200,"message":"OK","data":%v,"state":"%v","updated_time":%v}`
	OkResponseWithErrorCodeTemplate = `{"status":200,"message":"OK","data":%v,"state":"%v","updated_time":%v,"error_code":%v,"error_details":"%v"}`

	// TODO, this is should be retired
	TR181ResponseTemplate = `{"parameters":%v,"version":"%v"}`
)

// TODO: VersionHandler does not go through Middleware, hence the XpcResponseWriter cast will fail
// take no actions for now. Need to see if this causes errors
func SetAuditValue(w http.ResponseWriter, key string, value interface{}) {
	xw, ok := w.(*XpcResponseWriter)
	if !ok {
		// fields := make(log.Fields)
		// log.WithFields(fields).Error("internal error in openwebconfig.http.SetAuditValue() NotOK")
		return
	}
	fields := xw.Audit()
	fields[key] = value
}

func WriteByMarshal(w http.ResponseWriter, status int, o interface{}) {
	rbytes, err := json.Marshal(o)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		LogError(w, common.NewError(err))
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(status)
	w.Write(rbytes)
}

// helper function to wirte a json response into ResponseWriter
func WriteOkResponse(w http.ResponseWriter, data interface{}) {
	resp := common.HttpResponse{
		Status:  http.StatusOK,
		Message: http.StatusText(http.StatusOK),
		Data:    data,
	}
	SetAuditValue(w, "response", resp)
	WriteByMarshal(w, http.StatusOK, resp)
}

func WriteAcceptedResponse(w http.ResponseWriter) {
	resp := common.HttpResponse{
		Status:  http.StatusAccepted,
		Message: http.StatusText(http.StatusAccepted),
	}
	SetAuditValue(w, "response", resp)
	WriteByMarshal(w, http.StatusAccepted, resp)
}

func WriteOkResponseByTemplate(w http.ResponseWriter, dataStr string, state int, updatedTime int, errorCode *int, errorDetails *string) {
	stateText := common.States[state]
	s := "null"
	if len(dataStr) > 0 {
		s = dataStr
	}
	var rbytes []byte
	if errorCode != nil && *errorCode > 0 && errorDetails != nil && len(*errorDetails) > 0 {
		resp := common.HttpResponse{
			Status:       http.StatusOK,
			Message:      http.StatusText(http.StatusOK),
			State:        stateText,
			UpdatedTime:  updatedTime,
			ErrorCode:    *errorCode,
			ErrorDetails: *errorDetails,
		}
		SetAuditValue(w, "response", resp)
		rbytes = []byte(fmt.Sprintf(OkResponseWithErrorCodeTemplate, s, stateText, updatedTime, *errorCode, *errorDetails))
	} else {
		resp := common.HttpResponse{
			Status:      http.StatusOK,
			Message:     http.StatusText(http.StatusOK),
			State:       stateText,
			UpdatedTime: updatedTime,
		}
		SetAuditValue(w, "response", resp)
		rbytes = []byte(fmt.Sprintf(OkResponseTemplate, s, stateText, updatedTime))
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(rbytes)
}

// this is used to return default tr-181 payload while the cpe is not in the db
// TODO XPC-13444, need eval if changes here is needed
func WriteContentTypeAndResponse(w http.ResponseWriter, rbytes []byte, version string, contentType string) {
	w.Header().Set("Content-type", contentType)
	w.Header().Set("ETag", version)
	w.WriteHeader(http.StatusOK)
	w.Write(rbytes)
}

// helper function to write a failure json response into ResponseWriter
func WriteErrorResponse(w http.ResponseWriter, status int, err error) {
	errstr := ""
	if err != nil {
		errstr = err.Error()
	}
	resp := common.HttpErrorResponse{
		Status:  status,
		Message: http.StatusText(status),
		Errors:  errstr,
	}
	SetAuditValue(w, "response", resp)
	WriteByMarshal(w, status, resp)
}

func Error(w http.ResponseWriter, status int, err error) {
	// XPC-12489 calling WriteHeader() multiple times will cause errors in "content-type"
	//           ==> errors like 'superfluous response.WriteHeader call' in stderr
	switch status {
	case http.StatusNoContent, http.StatusNotModified, http.StatusForbidden:
		w.WriteHeader(status)
	default:
		WriteErrorResponse(w, status, err)
	}
}

func WriteResponseBytes(w http.ResponseWriter, rbytes []byte, statusCode int, vargs ...string) {
	if len(vargs) > 0 {
		w.Header().Set("Content-type", vargs[0])
	}
	w.WriteHeader(statusCode)
	w.Write(rbytes)
}

func WriteFactoryResetResponse(w http.ResponseWriter) {
	w.Header().Set("Content-type", common.MultipartContentType)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}
