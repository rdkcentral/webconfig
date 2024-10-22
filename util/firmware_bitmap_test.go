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
package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"gotest.tools/assert"
)

// bitmap is used to name int variables
// bitarray is used to name string variables

func TestPrettyBitarray(t *testing.T) {
	i := 8
	bs := PrettyBitarray(i)
	expected := "0000 0000 0000 0000 0000 0000 0000 1000"
	assert.Equal(t, bs, expected)

	j := 16777231
	bs = PrettyGroupBitarray(j)
	expected = "00000001 0000 0000 0000 0000 0000 1111"
	assert.Equal(t, bs, expected)
}

func TestParseRdkGroupBitarray(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777231,33554435,50331649,67108865,83886081,100663297,117440513,134217729"
	expectedMap := map[int]int{
		1: 15,
		2: 3,
		3: 1,
		4: 1,
		5: 1,
		6: 1,
		7: 1,
		8: 1,
	}

	sids := strings.Split(rdkSupportedDocsHeaderStr, ",")

	for _, sid := range sids {
		groupId, groupBitmap, err := ParseFirmwareGroupBitarray(sid)
		assert.NilError(t, err)
		// assert.Equal(t, groupId, 1)

		expectedBitmap, ok := expectedMap[groupId]
		assert.Assert(t, ok)
		assert.Equal(t, groupBitmap, expectedBitmap)
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)

	// bs := PrettyGroupBitarray(cpeBitmap)

	// build expected
	expectedEnabled := map[string]bool{
		"portforwarding":  true,
		"lan":             true,
		"wan":             true,
		"macbinding":      true,
		"hotspot":         false,
		"bridge":          false,
		"privatessid":     true,
		"homessid":        true,
		"radio":           false,
		"moca":            true,
		"xdns":            true,
		"advsecurity":     true,
		"mesh":            true,
		"aker":            true,
		"telemetry":       true,
		"statusreport":    false,
		"trafficreport":   false,
		"interfacereport": false,
		"radioreport":     false,
	}

	for subdocId, enabled := range expectedEnabled {
		parsedEnabled := IsSubdocSupported(cpeBitmap, subdocId)
		assert.Equal(t, parsedEnabled, enabled)
	}
}

func TestParseCustomizedGroupBitarray(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777231,33554435,50331649,67108865,83886081,100663297,117440513,134217729"
	sids := strings.Split(rdkSupportedDocsHeaderStr, ",")

	// build expected
	expectedEnabled := map[string]bool{
		"portforwarding":  true,
		"lan":             true,
		"wan":             true,
		"macbinding":      true,
		"hotspot":         false,
		"bridge":          false,
		"privatessid":     true,
		"homessid":        true,
		"radio":           false,
		"moca":            true,
		"xdns":            true,
		"advsecurity":     true,
		"mesh":            true,
		"aker":            true,
		"telemetry":       true,
		"statusreport":    false,
		"trafficreport":   false,
		"interfacereport": false,
		"radioreport":     false,
		"telcovoip":       false,
		"telcovoice":      false,
		"wanmanager":      false,
		"voiceservice":    false,
	}

	newGroup1Bitarray := "00000001 0000 0000 0000 0000 0011 0011"
	group1Bitmap, err := BitarrayToBitmap(newGroup1Bitarray)
	assert.NilError(t, err)
	sids[0] = fmt.Sprintf("%v", group1Bitmap)
	expectedEnabled["wan"] = false
	expectedEnabled["macbinding"] = false
	expectedEnabled["hotspot"] = true
	expectedEnabled["bridge"] = true

	newGroup2Bitarray := "00000010 0000 0000 0000 0000 0000 0110"
	group2Bitmap, err := BitarrayToBitmap(newGroup2Bitarray)
	assert.NilError(t, err)
	sids[1] = fmt.Sprintf("%v", group2Bitmap)
	expectedEnabled["privatessid"] = false
	expectedEnabled["radio"] = true

	rdkSupportedDocsHeaderStr = strings.Join(sids, ",")
	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	for subdocId, enabled := range expectedEnabled {
		parsedEnabled := IsSubdocSupported(cpeBitmap, subdocId)
		assert.Equal(t, parsedEnabled, enabled)
	}

	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocIds := []string{}
	for k, v := range expectedEnabled {
		if v {
			supportedSubdocIds = append(supportedSubdocIds, k)
		}
	}
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)

	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseTelcovoipAndWanmanager(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777231,33554433,67108865,83886081,100663297,117440513,134217729,184549377,201326593"
	sids := strings.Split(rdkSupportedDocsHeaderStr, ",")

	// build expected
	supportedSubdocIds := []string{
		"portforwarding",
		"lan",
		"wan",
		"macbinding",
		"privatessid",
		"xdns",
		"advsecurity",
		"mesh",
		"aker",
		"telemetry",
		"telcovoip",
		"wanmanager",
	}

	rdkSupportedDocsHeaderStr = strings.Join(sids, ",")
	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	for _, subdocId := range supportedSubdocIds {
		assert.Assert(t, IsSubdocSupported(cpeBitmap, subdocId))
	}
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestBitmapParsing(t *testing.T) {
	// clearn the wan bit
	newBitmap := 16777231 & ^(1 << 2)
	rdkSupportedDocsHeaderStr := fmt.Sprintf("%v,33554435,50331649,67108865,83886081,100663297,117440513,134217729", newBitmap)

	// build expected
	supportedSubdocIds := []string{
		"portforwarding",
		"lan",
		"macbinding",
		"privatessid",
		"homessid",
		"moca",
		"xdns",
		"advsecurity",
		"mesh",
		"aker",
		"telemetry",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseVoiceService(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729,218103809"

	supportedSubdocIds := []string{
		"portforwarding",
		"lan",
		"wan",
		"macbinding",
		"hotspot",
		"privatessid",
		"homessid",
		"moca",
		"xdns",
		"advsecurity",
		"mesh",
		"aker",
		"telemetry",
		"voiceservice",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestManualBitmap(t *testing.T) {
	for i := 0; i < 10; i++ {
		bitmap := RandomInt(40000)
		parsedSupportedMap := GetSupportedMap(bitmap)
		revBitmap := GetBitmapFromSupportedMap(parsedSupportedMap)
		assert.Equal(t, bitmap, revBitmap)
	}
}

func TestParseSupportedDocsWithNewGroups(t *testing.T) {
	cellularBitGroupId := 14
	xBitValue := (cellularBitGroupId << 24) + 1
	ss := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729,218103809,234881025"
	rdkSupportedDocsHeaderStr := fmt.Sprintf("%v,%v", ss, xBitValue)

	// build expected
	supportedSubdocIds := []string{
		"portforwarding",
		"lan",
		"wan",
		"macbinding",
		"hotspot",
		"privatessid",
		"homessid",
		"moca",
		"xdns",
		"advsecurity",
		"mesh",
		"aker",
		"telemetry",
		"voiceservice",
		"cellularconfig",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseSupportedDocsHeaderWithSomeLTEGroups(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777231,33554435,67108865,100663297,117440513,134217729,201326595,234881025"

	// build expected
	supportedSubdocIds := []string{
		"aker",
		"cellularconfig",
		"homessid",
		"lan",
		"macbinding",
		"mesh",
		"portforwarding",
		"privatessid",
		"telemetry",
		"wan",
		"wanfailover",
		"wanmanager",
		"xdns",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseSupportedDocsHeaderWithTelcovoice(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777231,33554433,67108865,83886081,100663297,117440513,134217729,184549378,201326595"

	// build expected
	supportedSubdocIds := []string{
		"advsecurity",
		"aker",
		"lan",
		"macbinding",
		"mesh",
		"portforwarding",
		"privatessid",
		"telcovoice",
		"telemetry",
		"wan",
		"wanfailover",
		"wanmanager",
		"xdns",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseSupportedDocsHeaderWithGwfailover(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729,201326594,218103809,251658241"

	// build expected
	supportedSubdocIds := []string{
		"advsecurity",
		"aker",
		"gwfailover",
		"homessid",
		"hotspot",
		"lan",
		"macbinding",
		"mesh",
		"moca",
		"portforwarding",
		"privatessid",
		"telemetry",
		"voiceservice",
		"wan",
		"wanfailover",
		"xdns",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseSupportedDocsHeaderWithPrioritizedMacs(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729,201326594,251658241,268435457"

	// build expected
	supportedSubdocIds := []string{
		"advsecurity",
		"aker",
		"gwfailover",
		"homessid",
		"hotspot",
		"lan",
		"macbinding",
		"mesh",
		"moca",
		"portforwarding",
		"privatessid",
		"telemetry",
		"wan",
		"wanfailover",
		"xdns",
		"prioritizedmacs",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseSupportedDocsHeaderWithPrioritizedMacsAndConnectedbuilding(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777311,33554435,50331649,67108865,83886081,100663297,117440513,134217729,201326594,218103809,251658241,268435457,285212673"

	// build expected
	supportedSubdocIds := []string{
		"advsecurity",
		"aker",
		"gwfailover",
		"homessid",
		"hotspot",
		"lan",
		"macbinding",
		"mesh",
		"moca",
		"portforwarding",
		"privatessid",
		"telemetry",
		"voiceservice",
		"wan",
		"wanfailover",
		"xdns",
		"prioritizedmacs",
		"connectedbuilding",
		"lldqoscontrol",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseSupportedDocsHeaderClienttosteeringprofile(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777311,33554435,50331649,67108865,83886081,100663299,117440513,134217729,201326594,218103809,251658241,268435457,285212673"

	// build expected
	supportedSubdocIds := []string{
		"advsecurity",
		"aker",
		"gwfailover",
		"homessid",
		"hotspot",
		"lan",
		"macbinding",
		"mesh",
		"moca",
		"portforwarding",
		"privatessid",
		"telemetry",
		"voiceservice",
		"wan",
		"wanfailover",
		"xdns",
		"prioritizedmacs",
		"connectedbuilding",
		"lldqoscontrol",
		"clienttosteeringprofile",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseSupportedDocsHeaderRfc(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777311,33554435,50331649,67108865,83886081,100663299,117440513,134217735,201326594,218103809,251658241,268435457,285212673"

	// build expected
	supportedSubdocIds := []string{
		"advsecurity",
		"aker",
		"gwfailover",
		"homessid",
		"hotspot",
		"lan",
		"macbinding",
		"mesh",
		"moca",
		"portforwarding",
		"privatessid",
		"telemetry",
		"voiceservice",
		"wan",
		"wanfailover",
		"xdns",
		"prioritizedmacs",
		"connectedbuilding",
		"lldqoscontrol",
		"clienttosteeringprofile",
		"defaultrfc",
		"rfc",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseSupportedDocsHeaderHcm(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777311,33554435,50331649,67108865,83886081,100663359,117440513,134217729,201326594,218103809,251658241,268435457,285212673"

	// build expected
	supportedSubdocIds := []string{
		"advsecurity",
		"aker",
		"gwfailover",
		"homessid",
		"hotspot",
		"lan",
		"macbinding",
		"mesh",
		"moca",
		"portforwarding",
		"privatessid",
		"telemetry",
		"voiceservice",
		"wan",
		"wanfailover",
		"xdns",
		"prioritizedmacs",
		"connectedbuilding",
		"lldqoscontrol",
		"clienttosteeringprofile",
		"meshsteeringprofiles",
		"wifistatsconfig",
		"mwoconfigs",
		"interference",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}

func TestParseSupportedDocsHeaderWebui(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777695,33554435,50331649,67108865,83886081,100663359,117440513,134217729,201326594,218103809,251658241,268435457,285212673"

	// build expected
	supportedSubdocIds := []string{
		"advsecurity",
		"aker",
		"gwfailover",
		"homessid",
		"hotspot",
		"lan",
		"macbinding",
		"mesh",
		"moca",
		"portforwarding",
		"privatessid",
		"telemetry",
		"voiceservice",
		"wan",
		"wanfailover",
		"xdns",
		"prioritizedmacs",
		"connectedbuilding",
		"lldqoscontrol",
		"clienttosteeringprofile",
		"meshsteeringprofiles",
		"wifistatsconfig",
		"mwoconfigs",
		"interference",
		"xmspeedboost",
		"webui",
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	supportedSubdocMap := common.BuildSupportedSubdocMapWithDefaults(supportedSubdocIds)
	assert.DeepEqual(t, parsedSupportedMap, supportedSubdocMap)
}
