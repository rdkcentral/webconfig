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
	OkResponseTemplate    = `{"status":200,"message":"OK","data":%v}`
	TR181ResponseTemplate = `{"parameters":%v,"version":"%v"}`
)

func writeByMarshal(w http.ResponseWriter, r *http.Request, status int, o interface{}) {
	if rbytes, err := json.Marshal(o); err == nil {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(status)
		w.Write(rbytes)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		LogError(w, r, err)
	}
}

//helper function to wirte a json response into ResponseWriter
func WriteOkResponse(w http.ResponseWriter, r *http.Request, data interface{}) {
	resp := common.HttpResponse{
		Status:  http.StatusOK,
		Message: http.StatusText(http.StatusOK),
		Data:    data,
	}
	writeByMarshal(w, r, http.StatusOK, resp)
}

func WriteOkResponseByTemplate(w http.ResponseWriter, r *http.Request, dataStr string) {
	rbytes := []byte(fmt.Sprintf(OkResponseTemplate, dataStr))
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(rbytes)
}

func WriteTR181Response(w http.ResponseWriter, r *http.Request, params string, version string) {
	w.Header().Set("Content-type", "application/json")
	w.Header().Set("ETag", version)
	w.WriteHeader(http.StatusOK)
	rbytes := []byte(fmt.Sprintf(TR181ResponseTemplate, params, version))
	w.Write(rbytes)
}

// this is used to return default tr-181 payload while the cpe is not in the db
func WriteContentTypeAndResponse(w http.ResponseWriter, r *http.Request, rbytes []byte, version string, contentType string) {
	w.Header().Set("Content-type", contentType)
	w.Header().Set("ETag", version)
	w.WriteHeader(http.StatusOK)
	w.Write(rbytes)
}

//helper function to write a failure json response into ResponseWriter
func WriteErrorResponse(w http.ResponseWriter, r *http.Request, status int, err error) {
	errstr := ""
	if err != nil {
		errstr = err.Error()
	}
	resp := common.HttpErrorResponse{
		Status:  status,
		Message: http.StatusText(status),
		Errors:  errstr,
	}
	writeByMarshal(w, r, status, resp)
}

func Error(w http.ResponseWriter, r *http.Request, status int, err error) {
	switch status {
	case http.StatusNoContent, http.StatusNotModified, http.StatusForbidden:
		w.WriteHeader(status)
	default:
		WriteErrorResponse(w, r, status, err)
	}
}

func WriteResponseBytes(w http.ResponseWriter, r *http.Request, rbytes []byte, statusCode int, vargs ...string) {
	if len(vargs) > 0 {
		w.Header().Set("Content-type", vargs[0])
	}
	w.WriteHeader(statusCode)
	w.Write(rbytes)
}

func WriteFactoryResetResponse(w http.ResponseWriter) {
	w.Header().Set("Content-type", MultipartContentType)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}
