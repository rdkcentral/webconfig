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

type XpcResponseWriter struct {
	http.ResponseWriter
	status         int
	length         int
	response       string
	startTime      time.Time
	body           string
	token          string
	audit          log.Fields
	bodyObfuscated bool
}

func (w *XpcResponseWriter) String() string {
	ret := fmt.Sprintf("status=%v, length=%v, response=%v, startTime=%v, audit=%v",
		w.status, w.length, w.response, w.startTime, w.audit)
	return ret
}

func NewXpcResponseWriter(w http.ResponseWriter, vargs ...interface{}) *XpcResponseWriter {
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

	return &XpcResponseWriter{
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
func (w *XpcResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *XpcResponseWriter) Write(b []byte) (int, error) {
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
func (w *XpcResponseWriter) Status() int {
	return w.status
}

func (w *XpcResponseWriter) Response() string {
	return w.response
}

func (w *XpcResponseWriter) StartTime() time.Time {
	return w.startTime
}

func (w *XpcResponseWriter) AuditId() string {
	return w.AuditData("audit_id")
}

func (w *XpcResponseWriter) Body() string {
	return w.AuditData("body")
}

func (w *XpcResponseWriter) SetBody(body string) {
	w.SetAuditData("body", body)
}

func (w *XpcResponseWriter) Token() string {
	return w.token
}

func (w *XpcResponseWriter) TraceId() string {
	return w.AuditData("trace_id")
}

func (w *XpcResponseWriter) Audit() log.Fields {
	// return w.audit
	out := log.Fields{}
	for k, v := range w.audit {
		if k == "body" && w.bodyObfuscated {
			out[k] = "****"
		} else {
			out[k] = v
		}
	}
	return out
}

func (w *XpcResponseWriter) AuditData(k string) string {
	itf := w.audit[k]
	if itf != nil {
		return itf.(string)
	}
	return ""
}

func (w *XpcResponseWriter) SetAuditData(k string, v interface{}) {
	w.audit[k] = v
}

func (w *XpcResponseWriter) SetBodyObfuscated(obfuscated bool) {
	w.bodyObfuscated = obfuscated
}
