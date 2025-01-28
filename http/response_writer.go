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
	"time"

	"github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"
)

var (
	ObfuscatedMap = map[string]string{
		"msgpack": "****",
	}
	BadJsonResponseMap = map[string]string{
		"bad_json": "****",
	}
	EmptyMap = map[string]string{}
)

type XResponseWriter struct {
	http.ResponseWriter
	status    int
	length    int
	response  string
	startTime time.Time
	bodyBytes []byte
	token     string
	audit     log.Fields
	partnerId string
}

func (w *XResponseWriter) String() string {
	ret := fmt.Sprintf("status=%v, length=%v, response=%v, startTime=%v, audit=%v",
		w.status, w.length, w.response, w.startTime, w.audit)
	return ret
}

func NewXResponseWriter(w http.ResponseWriter, vargs ...interface{}) *XResponseWriter {
	var audit log.Fields
	var startTime time.Time
	var token string

	for _, v := range vargs {
		switch ty := v.(type) {
		case time.Time:
			startTime = ty
		case log.Fields:
			audit = ty
		case string:
			token = ty
		}
	}

	if audit == nil {
		audit = make(log.Fields)
	}

	return &XResponseWriter{
		ResponseWriter: w,
		status:         0,
		length:         0,
		response:       "",
		startTime:      startTime,
		token:          token,
		audit:          audit,
	}
}

// interface/override
func (w *XResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *XResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	if err != nil {
		return n, common.NewError(err)
	}
	w.length += n
	w.response = string(b)
	return n, nil
}

// get/set
func (w *XResponseWriter) Status() int {
	return w.status
}

func (w *XResponseWriter) Response() string {
	return w.response
}

func (w *XResponseWriter) StartTime() time.Time {
	return w.startTime
}

func (w *XResponseWriter) AuditId() string {
	return w.AuditData("audit_id")
}

func (w *XResponseWriter) BodyBytes() []byte {
	return w.bodyBytes
}

func (w *XResponseWriter) SetBodyBytes(bbytes []byte) {
	w.bodyBytes = bbytes
}

func (w *XResponseWriter) Token() string {
	return w.token
}

func (w *XResponseWriter) SetToken(token string) {
	w.token = token
}

func (w *XResponseWriter) TraceId() string {
	return w.AuditData("trace_id")
}

func (w *XResponseWriter) Audit() log.Fields {
	return w.audit
}

func (w *XResponseWriter) AuditData(k string) string {
	itf := w.audit[k]
	if itf != nil {
		return itf.(string)
	}
	return ""
}

func (w *XResponseWriter) SetAuditData(k string, v interface{}) {
	w.audit[k] = v
}

func (w *XResponseWriter) PartnerId() string {
	return w.partnerId
}

func (w *XResponseWriter) SetPartnerId(partnerId string) {
	w.partnerId = partnerId
	w.audit["partner"] = partnerId
}

func (w *XResponseWriter) ReqMoracideTags() map[string]string {
	itf, ok := w.audit["req_moracide_tags"]
	if !ok {
		return nil
	}
	return itf.(map[string]string)
}

func (w *XResponseWriter) RespMoracideTags() map[string]string {
	itf, ok := w.audit["resp_moracide_tags"]
	if !ok {
		return nil
	}
	return itf.(map[string]string)
}
