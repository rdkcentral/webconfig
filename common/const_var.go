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
package common

const (
	Deployed = iota + 1
	PendingDownload
	InDeployment
	Failure
)

var (
	States = [5]string{
		"",
		"deployed",
		"pending download",
		"in deployment",
		"failure",
	}
)

const (
	LoggingTimeFormat = "2006-01-02 15:04:05.000"
	PokeBodyTemplate  = `{"parameters":[{"dataType":0,"name":"Device.X_RDK_WebConfig.ForceSync","value":"%s"}]}`
)

var (
	BinaryVersion   = ""
	BinaryBranch    = ""
	BinaryBuildTime = ""
	OpenLibVersion  = ""

	DefaultIgnoredHeaders = []string{
		"Accept",
		"User-Agent",
		"Authorization",
		"Content-Type",
		"Content-Length",
		"Accept-Encoding",
		"X-B3-Sampled",
		"X-B3-Spanid",
		"X-B3-Traceid",
		"X-Envoy-Decorator-Operation",
		"X-Envoy-External-Address",
		"X-Envoy-Peer-Metadata",
		"X-Envoy-Peer-Metadata-Id",
		"X-Forwarded-Proto",
	}
)

const (
	HeaderContentType                = "Content-Type"
	HeaderApplicationJson            = "application/json"
	HeaderApplicationMsgpack         = "application/msgpack"
	HeaderEtag                       = "Etag"
	HeaderIfNoneMatch                = "If-None-Match"
	HeaderFirmwareVersion            = "X-System-Firmware-Version"
	HeaderSupportedDocs              = "X-System-Supported-Docs"
	HeaderSupplementaryService       = "X-System-SupplementaryService-Sync"
	HeaderModelName                  = "X-System-Model-Name"
	HeaderProfileVersion             = "X-System-Telemetry-Profile-Version"
	HeaderPartnerID                  = "X-System-PartnerID"
	HeaderAccountID                  = "X-System-AccountID"
	HeaderProductClass               = "X-System-Product-Class"
	HeaderUserAgent                  = "User-Agent"
	HeaderSchemaVersion              = "X-System-Schema-Version"
	HeaderMetricsAgent               = "X-Metrics-Agent"
	HeaderStoreUpstreamResponse      = "X-Store-Upstream-Response"
	HeaderSubdocumentVersion         = "X-Subdocument-Version"
	HeaderSubdocumentState           = "X-Subdocument-State"
	HeaderSubdocumentUpdatedTime     = "X-Subdocument-Updated-Time"
	HeaderSubdocumentErrorCode       = "X-Subdocument-Error-Code"
	HeaderSubdocumentErrorDetails    = "X-Subdocument-Error-Details"
	HeaderSubdocumentExpiry          = "X-Subdocument-Expiry"
	HeaderSubdocumentOldState        = "X-Subdocument-Old-State"
	HeaderSubdocumentMetricsAgent    = "X-Subdocument-Metrics-Agent"
	HeaderDeviceId                   = "Device-Id"
	HeaderDocName                    = "Doc-Name"
	HeaderUpstreamNewBitmap          = "X-Upstream-New-Bitmap"
	HeaderUpstreamNewFirmwareVersion = "X-Upstream-New-Firmware-Version"
	HeaderUpstreamNewModelName       = "X-Upstream-New-Model-Name"
	HeaderUpstreamNewPartnerId       = "X-Upstream-New-Partner-Id"
	HeaderUpstreamNewSchemaVersion   = "X-Upstream-New-Schema-Version"
	HeaderUpstreamOldBitmap          = "X-Upstream-Old-Bitmap"
	HeaderUpstreamOldFirmwareVersion = "X-Upstream-Old-Firmware-Version"
	HeaderUpstreamOldModelName       = "X-Upstream-Old-Model-Name"
	HeaderUpstreamOldPartnerId       = "X-Upstream-Old-Partner-Id"
	HeaderUpstreamOldSchemaVersion   = "X-Upstream-Old-Schema-Version"
	HeaderAuthorization              = "Authorization"
	HeaderAuditid                    = "X-Auditid"
	HeaderTransactionId              = "Transaction-Id"
	HeaderReqUrl                     = "X-Req-Url"
	HeaderWanMac                     = "X-System-Wan-Mac"
	HeaderSourceAppName              = "X-Source-App-Name"
	HeaderTraceparent                = "Traceparent"
	HeaderTracestate                 = "Tracestate"
	HeaderContentLength              = "Content-Length"
	HeaderRefSubdocumentVersion      = "X-Refsubdocument-Version"
	HeaderUpstreamResponse           = "X-Upstream-Response"
)

const (
	SkipDbUpdate = "skip-db-update"
)

var (
	SupportedPokeDocs   = []string{"primary", "telemetry", "root"}
	SupportedPokeRoutes = []string{"mqtt"}
)

var (
	CRLFCRLF = []byte("\r\n\r\n")
	CRLF     = []byte("\r\n")
)

const (
	RouteMqtt = "mqtt"
	RouteHttp = "http"
)

const (
	RootDocumentEquals = iota
	RootDocumentVersionOnlyChanged
	RootDocumentMetaChanged
	RootDocumentMissing
)
