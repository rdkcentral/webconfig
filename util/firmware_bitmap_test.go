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
	t.Logf("i=%v\n", i)
	bs := PrettyBitarray(i)
	t.Logf("%v\n\n", bs)
	expected := "0000 0000 0000 0000 0000 0000 0000 1000"
	assert.Equal(t, bs, expected)

	j := 16777231
	t.Logf("j=%v\n", j)
	bs = PrettyGroupBitarray(j)
	t.Logf("%v\n\n", bs)
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
	t.Logf("cpeBitmap=%v\n\n", cpeBitmap)

	bs := PrettyGroupBitarray(cpeBitmap)
	t.Logf("%v\n\n", bs)

	// build expected
	expectedEnabled := map[string]bool{
		"portforwarding":  true,
		"lan":             true,
		"wan":             true,
		"macbinding":      true,
		"xfinity":         false,
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
	t.Logf("\n\nsids=%v\n", sids)

	// build expected
	expectedEnabled := map[string]bool{
		"portforwarding":  true,
		"lan":             true,
		"wan":             true,
		"macbinding":      true,
		"xfinity":         false,
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

	newGroup1Bitarray := "00000001 0000 0000 0000 0000 0011 0011"
	group1Bitmap, err := BitarrayToBitmap(newGroup1Bitarray)
	assert.NilError(t, err)
	sids[0] = fmt.Sprintf("%v", group1Bitmap)
	expectedEnabled["wan"] = false
	expectedEnabled["macbinding"] = false
	expectedEnabled["xfinity"] = true
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
	assert.DeepEqual(t, parsedSupportedMap, expectedEnabled)
}
