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
package tracing

import (
	"net/http"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// XpcTrace is a carrier/baggage struct to extract data from spans, request headers for usage later
// Store the trace in ctx for easy retrieval.
// Ideal place to store it is ofc, xw
// But because of legacy reasons, xw is not always available in the API flow
type XpcTrace struct {
	ReqMoracideTag string
	// The response moracide tags are stored in xw.audit

	// traceparent, tracestate can be set as req headers, may be extracted from otel spans
	// These need to be propagated to any http calls we make
	// Order of priority; use the value extracted from otel span;
	// if no otel span as well, use the value in req headers (Note: this will create islands as both
	// the app and its children will have the same tracestate
	// Otherwise, nothing will be passed to the child http calls, creating islands
	// If any source is found, then it will be propagated to all child http calls
	// TODO; also add this to Kafka headers, SNS message attributes
	otelTraceparent string
	otelTracestate  string
	ReqTraceparent  string
	ReqTracestate   string
	OutTraceparent  string
	OutTracestate   string

	// At the end of API flow, add the status code to OtelSpan; add the Moracide tags to the spans
	otelSpan oteltrace.Span

	// These are not useful as of now, just set them for the sake of completion and future
	AuditID      string
	MoneyTrace   string
	ReqUserAgent string
	OutUserAgent string

	TraceID string // use the value in outTraceparent, otherwise MoneyTrace
}

// NewXpcTrace extracts traceparent, tracestate, moracideTags from otel spans or reqs
func NewXpcTrace(xpcTracer *XpcTracer, r *http.Request) *XpcTrace {
	var xpcTrace XpcTrace
	extractParamsFromReq(r, &xpcTrace, xpcTracer.AppName())

	if xpcTracer.OtelEnabled {
		otelExtractParamsFromSpan(r.Context(), &xpcTrace)
	}

	return &xpcTrace
}

func SetSpanStatusCode(fields log.Fields) {
	var xpcTrace *XpcTrace
	if tmp, ok := fields["xpc_trace"]; ok {
		xpcTrace = tmp.(*XpcTrace)
	}
	if xpcTrace == nil {
		// Something went wrong, cannot instrument this span
		log.Error("instrumentation error, no trace info")
		return
	}
	if xpcTrace.otelSpan != nil {
		if tmp, ok := fields["status"]; ok {
			statusCode := tmp.(int)
			otelSetStatusCode(xpcTrace.otelSpan, statusCode)
		}
	}
}

func SetSpanMoracideTags(fields log.Fields, moracideTagPrefix string) {
	var xpcTrace *XpcTrace
	if tmp, ok := fields["xpc_trace"]; ok {
		xpcTrace = tmp.(*XpcTrace)
	}
	if xpcTrace == nil {
		// Something went wrong, cannot instrument this span
		log.Error("instrumentation error, cannot set moracide tags, no trace info")
		return
	}

	moracide := util.FieldsGetString(fields, "resp_moracide_tag")
	if len(moracide) == 0 {
		moracide = util.FieldsGetString(fields, "req_moracide_tag")
	}

	if xpcTrace.otelSpan != nil && len(moracide) > 0 {
		xpcTrace.otelSpan.SetAttributes(attribute.String(common.HeaderMoracide, moracide))
	}
}

func extractParamsFromReq(r *http.Request, xpcTrace *XpcTrace, serviceName string) {
	xpcTrace.ReqTraceparent = r.Header.Get(common.HeaderTraceparent)
	xpcTrace.ReqTracestate = r.Header.Get(common.HeaderTracestate)
	xpcTrace.OutTraceparent = xpcTrace.ReqTraceparent
	xpcTrace.OutTracestate = xpcTrace.ReqTracestate
	xpcTrace.ReqUserAgent = r.Header.Get(UserAgentHeader)
	xpcTrace.ReqMoracideTag = r.Header.Get(common.HeaderMoracide)
	if ss := r.Header.Get(common.HeaderCanary); ss == "true" {
		if len(xpcTrace.ReqMoracideTag) > 0 {
			xpcTrace.ReqMoracideTag += "," + serviceName
		} else {
			xpcTrace.ReqMoracideTag = serviceName
		}
	}
}
