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
	expectedEnabled["wanfailover"] = false
	expectedEnabled["cellularconfig"] = false
	expectedEnabled["gwfailover"] = false
	assert.DeepEqual(t, parsedSupportedMap, expectedEnabled)
}

func TestParseTelcovoipAndWanmanager(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777231,33554433,67108865,83886081,100663297,117440513,134217729,184549377,201326593"
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
		"homessid":        false,
		"radio":           false,
		"moca":            false,
		"xdns":            true,
		"advsecurity":     true,
		"mesh":            true,
		"aker":            true,
		"telemetry":       true,
		"statusreport":    false,
		"trafficreport":   false,
		"interfacereport": false,
		"radioreport":     false,
		"telcovoip":       true,
		"telcovoice":      false,
		"wanmanager":      true,
		"voiceservice":    false,
	}

	rdkSupportedDocsHeaderStr = strings.Join(sids, ",")
	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	for subdocId, enabled := range expectedEnabled {
		parsedEnabled := IsSubdocSupported(cpeBitmap, subdocId)
		assert.Equal(t, parsedEnabled, enabled)
	}

	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	expectedEnabled["wanfailover"] = false
	expectedEnabled["cellularconfig"] = false
	expectedEnabled["gwfailover"] = false
	assert.DeepEqual(t, parsedSupportedMap, expectedEnabled)
}

func TestBitmapParsing(t *testing.T) {
	// clearn the wan bit
	newBitmap := 16777231 & ^(1 << 2)
	rdkSupportedDocsHeaderStr := fmt.Sprintf("%v,33554435,50331649,67108865,83886081,100663297,117440513,134217729", newBitmap)

	// build expected
	expectedEnabled := map[string]bool{
		"portforwarding":  true,
		"lan":             true,
		"wan":             false,
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

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	for subdocId, enabled := range expectedEnabled {
		parsedEnabled := IsSubdocSupported(cpeBitmap, subdocId)
		assert.Equal(t, parsedEnabled, enabled)
	}

	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	expectedEnabled["wanfailover"] = false
	expectedEnabled["cellularconfig"] = false
	expectedEnabled["gwfailover"] = false
	assert.DeepEqual(t, parsedSupportedMap, expectedEnabled)
}

func TestParseVoiceService(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729,218103809"
	// sids := strings.Split(rdkSupportedDocsHeaderStr, ",")

	// build expected
	expectedEnabled := map[string]bool{
		"portforwarding":  true,
		"lan":             true,
		"wan":             true,
		"macbinding":      true,
		"hotspot":         true,
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
		"voiceservice":    true,
	}

	// rdkSupportedDocsHeaderStr = strings.Join(sids, ",")
	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	for subdocId, enabled := range expectedEnabled {
		parsedEnabled := IsSubdocSupported(cpeBitmap, subdocId)
		assert.Equal(t, parsedEnabled, enabled)
	}

	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	expectedEnabled["wanfailover"] = false
	expectedEnabled["cellularconfig"] = false
	expectedEnabled["gwfailover"] = false
	assert.DeepEqual(t, parsedSupportedMap, expectedEnabled)
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
	expectedEnabled := map[string]bool{
		"portforwarding":  true,
		"lan":             true,
		"wan":             true,
		"macbinding":      true,
		"hotspot":         true,
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
		"voiceservice":    true,
		"cellularconfig":  true,
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	for subdocId, enabled := range expectedEnabled {
		parsedEnabled := IsSubdocSupported(cpeBitmap, subdocId)
		assert.Equal(t, parsedEnabled, enabled)
	}

	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	expectedEnabled["wanfailover"] = false
	expectedEnabled["gwfailover"] = false
	assert.DeepEqual(t, parsedSupportedMap, expectedEnabled)
}

func TestParseSupportedDocsHeaderWithSomeLTEGroups(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777231,33554435,67108865,100663297,117440513,134217729,201326595,234881025"

	// build expected
	expectedEnabled := map[string]bool{
		"advsecurity":     false,
		"aker":            true,
		"bridge":          false,
		"cellularconfig":  true,
		"homessid":        true,
		"hotspot":         false,
		"interfacereport": false,
		"lan":             true,
		"macbinding":      true,
		"mesh":            true,
		"moca":            false,
		"portforwarding":  true,
		"privatessid":     true,
		"radio":           false,
		"radioreport":     false,
		"statusreport":    false,
		"telcovoip":       false,
		"telcovoice":      false,
		"telemetry":       true,
		"trafficreport":   false,
		"voiceservice":    false,
		"wan":             true,
		"wanfailover":     true,
		"wanmanager":      true,
		"xdns":            true,
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	for subdocId, enabled := range expectedEnabled {
		parsedEnabled := IsSubdocSupported(cpeBitmap, subdocId)
		assert.Equal(t, parsedEnabled, enabled)
	}

	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	expectedEnabled["gwfailover"] = false
	assert.DeepEqual(t, parsedSupportedMap, expectedEnabled)
}

func TestParseSupportedDocsHeaderWithTelcovoice(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777231,33554433,67108865,83886081,100663297,117440513,134217729,184549378,201326595"

	// build expected
	expectedEnabled := map[string]bool{
		"advsecurity":     true,
		"aker":            true,
		"bridge":          false,
		"cellularconfig":  false,
		"homessid":        false,
		"hotspot":         false,
		"interfacereport": false,
		"lan":             true,
		"macbinding":      true,
		"mesh":            true,
		"moca":            false,
		"portforwarding":  true,
		"privatessid":     true,
		"radio":           false,
		"radioreport":     false,
		"statusreport":    false,
		"telcovoip":       false,
		"telcovoice":      true,
		"telemetry":       true,
		"trafficreport":   false,
		"voiceservice":    false,
		"wan":             true,
		"wanfailover":     true,
		"wanmanager":      true,
		"xdns":            true,
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	for subdocId, enabled := range expectedEnabled {
		parsedEnabled := IsSubdocSupported(cpeBitmap, subdocId)
		assert.Equal(t, parsedEnabled, enabled)
	}

	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	expectedEnabled["gwfailover"] = false
	assert.DeepEqual(t, parsedSupportedMap, expectedEnabled)
}

func TestParseSupportedDocsHeaderWithGwfailover(t *testing.T) {
	rdkSupportedDocsHeaderStr := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729,201326594,218103809,251658241"

	// build expected
	expectedEnabled := map[string]bool{
		"advsecurity":     true,
		"aker":            true,
		"bridge":          false,
		"cellularconfig":  false,
		"gwfailover":      true,
		"homessid":        true,
		"hotspot":         true,
		"interfacereport": false,
		"lan":             true,
		"macbinding":      true,
		"mesh":            true,
		"moca":            true,
		"portforwarding":  true,
		"privatessid":     true,
		"radio":           false,
		"radioreport":     false,
		"statusreport":    false,
		"telcovoice":      false,
		"telcovoip":       false,
		"telemetry":       true,
		"trafficreport":   false,
		"voiceservice":    true,
		"wan":             true,
		"wanfailover":     true,
		"wanmanager":      false,
		"xdns":            true,
	}

	cpeBitmap, err := GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	for subdocId, enabled := range expectedEnabled {
		parsedEnabled := IsSubdocSupported(cpeBitmap, subdocId)
		assert.Equal(t, parsedEnabled, enabled)
	}

	parsedSupportedMap := GetSupportedMap(cpeBitmap)
	assert.DeepEqual(t, parsedSupportedMap, expectedEnabled)
}
