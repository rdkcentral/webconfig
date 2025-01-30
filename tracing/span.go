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
	"strings"

	"github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// XpcTrace is a carrier/baggage struct to extract data from spans, request headers for usage later
// Store the trace in ctx for easy retrieval.
// Ideal place to store it is ofc, xw
// But because of legacy reasons, xw is not always available in the API flow
type XpcTrace struct {
	// This is a bit of overengineering, but multiple tags are possible
	// e.g. X-Cl-Experiment-1, X-Cl-Experiment-xapproxy, X-Cl-Experiement-webconfig-25.1.1.1...
	// For every key found in either req or resp, an explicit value of true/false will be set as an otel attribute
	// or an otel span attribute
	ReqMoracideTags map[string]string // These are request headers prefixed with MoracideTagPrefix
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
	xpcTrace.ReqMoracideTags = make(map[string]string)

	extractParamsFromReq(r, &xpcTrace, xpcTracer.MoracideTagPrefix())

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

	moracideTags := make(map[string]string)

	if itf, ok := fields["req_moracide_tags"]; ok {
		reqMoracideTags := itf.(map[string]string)
		for key, val := range reqMoracideTags {
			moracideTags[key] = val
		}
	}

	if itf, ok := fields["resp_moracide_tags"]; ok {
		respMoracideTags := itf.(map[string]string)
		for key, val := range respMoracideTags {
			if val == "true" {
				moracideTags[key] = val
			}
		}
	}

	if xpcTrace.otelSpan != nil {
		for key, val := range moracideTags {
			xpcTrace.otelSpan.SetAttributes(attribute.String(key, val))
			log.Debugf("added otel span moracide tag key = %s, value = %s", key, val)
		}
	}
}

func extractParamsFromReq(r *http.Request, xpcTrace *XpcTrace, moracideTagPrefix string) {
	xpcTrace.ReqTraceparent = r.Header.Get(common.HeaderTraceparent)
	xpcTrace.ReqTracestate = r.Header.Get(common.HeaderTracestate)
	xpcTrace.OutTraceparent = xpcTrace.ReqTraceparent
	xpcTrace.OutTracestate = xpcTrace.ReqTracestate
	log.Debugf("Tracing: input traceparent : %s, tracestate : %s", xpcTrace.ReqTraceparent, xpcTrace.ReqTracestate)

	xpcTrace.ReqUserAgent = r.Header.Get(UserAgentHeader)

	// In future, -H 'X-Cl-Experiment-1', -H 'X-Cl-Experiment-oswebconfig'... OR 'X-Cl-Experiment-xapproxy_25.1.1.1' are all possible
	// So walk through all headers and collect any header that starts with this prefix
	moracideTagPrefix = strings.ToLower(moracideTagPrefix)
	for headerKey, headerVals := range r.Header {
		if strings.HasPrefix(strings.ToLower(headerKey), moracideTagPrefix) {
			if len(headerVals) > 1 {
				log.Debugf("Tracing: moracide tag key = %s, has multiple values = %+v", headerKey, headerVals)
			}
			val := "false"
			for _, v := range headerVals {
				if v == "true" {
					val = v
					break
				}
			}
			xpcTrace.ReqMoracideTags[headerKey] = val
			log.Debugf("Tracing: found moracide tag key = %s, val = %s, all vals = %+v", headerKey, val, headerVals)
		}
	}
}
