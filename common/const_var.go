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

const (
	LoggingTimeFormat = "2006-01-02 15:04:05.000"
)

var (
	BinaryVersion   = ""
	BinaryBranch    = ""
	BinaryBuildTime = ""

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
	HeaderIfNoneMatch     = "If-None-Match"
	HeaderFirmwareVersion = "X-System-Firmware-Version"
	HeaderSupportedDocs   = "X-System-Supported-Docs"
)

// header X-System-Supported-Docs
type BitMaskTuple struct {
	GroupBit int
	CpeBit   int
}

// The group based bitmaps will be merged into 1 cpe bitmap
// 1: []BitMaskTuple{ // group_id:
//    BitMaskTuple{1, 1},  // {"index_of_bit_from_lsb" for a group bitmap, "index_of_bit_from_lsb" for the cpe bitmap
//
var (
	SupportedDocsBitMaskMap = map[int][]BitMaskTuple{
		1: []BitMaskTuple{
			BitMaskTuple{1, 1},
			BitMaskTuple{2, 2},
			BitMaskTuple{3, 3},
			BitMaskTuple{4, 4},
			BitMaskTuple{5, 5},
			BitMaskTuple{6, 6},
		},
		2: []BitMaskTuple{
			BitMaskTuple{1, 7},
			BitMaskTuple{2, 8},
			BitMaskTuple{3, 9},
		},
		3: []BitMaskTuple{
			BitMaskTuple{1, 10},
		},
		4: []BitMaskTuple{
			BitMaskTuple{1, 11},
		},
		5: []BitMaskTuple{
			BitMaskTuple{1, 12},
		},
		6: []BitMaskTuple{
			BitMaskTuple{1, 13},
		},
		7: []BitMaskTuple{
			BitMaskTuple{1, 14},
		},
		8: []BitMaskTuple{
			BitMaskTuple{1, 15},
		},
		9: []BitMaskTuple{
			BitMaskTuple{1, 16},
			BitMaskTuple{2, 17},
		},
		10: []BitMaskTuple{
			BitMaskTuple{1, 18},
			BitMaskTuple{2, 19},
		},
	}
)

var (
	SubdocBitIndexMap = map[string]int{
		"portforwarding":  1,
		"lan":             2,
		"wan":             3,
		"macbinding":      4,
		"xfinity":         5,
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
	}
)
