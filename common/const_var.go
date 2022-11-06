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
	HeaderIfNoneMatch             = "If-None-Match"
	HeaderFirmwareVersion         = "X-System-Firmware-Version"
	HeaderSupportedDocs           = "X-System-Supported-Docs"
	HeaderSupplementaryService    = "X-System-SupplementaryService-Sync"
	HeaderModelName               = "X-System-Model-Name"
	HeaderProfileVersion          = "X-System-Telemetry-Profile-Version"
	HeaderPartnerID               = "X-System-PartnerID"
	HeaderAccountID               = "X-System-AccountID"
	HeaderUserAgent               = "User-Agent"
	HeaderSchemaVersion           = "X-System-Schema-Version"
	HeaderMetricsAgent            = "X-Metrics-Agent"
	HeaderStoreUpstreamResponse   = "X-Store-Upstream-Response"
	HeaderSubdocumentVersion      = "X-Subdocument-Version"
	HeaderSubdocumentState        = "X-Subdocument-State"
	HeaderSubdocumentUpdatedTime  = "X-Subdocument-Updated-Time"
	HeaderSubdocumentErrorCode    = "X-Subdocument-Error-Code"
	HeaderSubdocumentErrorDetails = "X-Subdocument-Error-Details"
	HeaderDeviceId                = "Device-Id"
	HeaderDocName                 = "Doc-Name"
)

// header X-System-Supported-Docs
type BitMaskTuple struct {
	GroupBit int
	CpeBit   int
}

// The group based bitmaps will be merged into 1 cpe bitmap
// 1: []BitMaskTuple{ // meta_group_id: defined by RDK
//
//	BitMaskTuple{1, 1},  // {"index_of_bit_from_lsb" for a group bitmap, "index_of_bit_from_lsb" for the cpe bitmap
var (
	SupportedDocsBitMaskMap = map[int][]BitMaskTuple{
		1: {
			{1, 1},
			{2, 2},
			{3, 3},
			{4, 4},
			{5, 5},
			{6, 6},
		},
		2: {
			{1, 7},
			{2, 8},
			{3, 9},
		},
		3: {
			{1, 10},
		},
		4: {
			{1, 11},
		},
		5: {
			{1, 12},
		},
		6: {
			{1, 13},
		},
		7: {
			{1, 14},
		},
		8: {
			{1, 15},
		},
		9: {
			{1, 16},
			{2, 17},
		},
		10: {
			{1, 18},
			{2, 19},
		},
		11: {
			{1, 20},
			{2, 25},
		},
		12: {
			{1, 21},
			{2, 23},
		},
		13: {
			{1, 22},
		},
		14: {
			{1, 24},
		},
	}
)

var (
	SubdocBitIndexMap = map[string]int{
		"portforwarding":  1,
		"lan":             2,
		"wan":             3,
		"macbinding":      4,
		"hotspot":         5,
		"bridge":          6,
		"privatessid":     7,
		"homessid":        8,
		"radio":           9,
		"moca":            10,
		"xdns":            11,
		"advsecurity":     12,
		"mesh":            13,
		"aker":            14,
		"telemetry":       15,
		"statusreport":    16,
		"trafficreport":   17,
		"interfacereport": 18,
		"radioreport":     19,
		"telcovoip":       20,
		"wanmanager":      21,
		"voiceservice":    22,
		"wanfailover":     23,
		"cellularconfig":  24,
		"telcovoice":      25,
	}
)

var (
	SupportedPokeDocs   = []string{"primary", "telemetry"}
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
