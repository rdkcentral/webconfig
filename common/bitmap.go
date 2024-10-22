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
			{7, 29}, // connectedbuilding
			{8, 35}, // xmspeedboost
			{9, 40}, // webui
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
			{1, 13}, // mesh
			{2, 31}, // clienttosteeringprofile
			{3, 36}, // meshsteeringprofiles
			{4, 37}, // wifistatsconfig
			{5, 38}, // mwoconfigs
			{6, 39}, // interference
			{7, 34}, // wifimotionsettings
		},
		7: {
			{1, 14},
		},
		8: {
			{1, 15},
			{2, 32},
			{3, 33},
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
		15: {
			{1, 26},
			{2, 27},
		},
		16: {
			{1, 28},
		},
		17: {
			{1, 30},
		},
	}
)

var (
	SubdocBitIndexMap = map[string]int{
		"portforwarding":          1,
		"lan":                     2,
		"wan":                     3,
		"macbinding":              4,
		"hotspot":                 5,
		"bridge":                  6,
		"privatessid":             7,
		"homessid":                8,
		"radio":                   9,
		"moca":                    10,
		"xdns":                    11,
		"advsecurity":             12,
		"mesh":                    13,
		"aker":                    14,
		"telemetry":               15,
		"statusreport":            16,
		"trafficreport":           17,
		"interfacereport":         18,
		"radioreport":             19,
		"telcovoip":               20,
		"wanmanager":              21,
		"voiceservice":            22,
		"wanfailover":             23,
		"cellularconfig":          24,
		"telcovoice":              25,
		"gwfailover":              26,
		"gwrestore":               27,
		"prioritizedmacs":         28,
		"connectedbuilding":       29,
		"lldqoscontrol":           30,
		"clienttosteeringprofile": 31,
		"defaultrfc":              32,
		"rfc":                     33,
		"wifimotionsettings":      34,
		"xmspeedboost":            35,
		"meshsteeringprofiles":    36,
		"wifistatsconfig":         37,
		"mwoconfigs":              38,
		"interference":            39,
		"webui":                   40,
	}
)

func GetDefaultSupportedSubdocMap() map[string]bool {
	m := make(map[string]bool)
	for k := range SubdocBitIndexMap {
		m[k] = false
	}
	return m
}

func BuildSupportedSubdocMapWithDefaults(supportedSubdocIds []string) map[string]bool {
	m := make(map[string]bool)
	for k := range SubdocBitIndexMap {
		m[k] = false
	}
	for _, s := range supportedSubdocIds {
		m[s] = true
	}
	return m
}
