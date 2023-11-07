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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestSupportedGroupsHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== step 1 test when no data ====
	cpeMac := util.GenerateRandomCpeMac()

	// call GET /supported_groups when no data
	sgUrl := fmt.Sprintf("/api/v1/device/%v/supported_groups", cpeMac)
	req, err := http.NewRequest("GET", sgUrl, nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// call GET /config to add supported-doc header
	configUrl := fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	rdkSupportedDocsHeaderStr := "16777231,33554435,50331649,67108865,83886081,100663297,117440513,134217729"
	req.Header.Set(common.HeaderSupportedDocs, rdkSupportedDocsHeaderStr)
	fwVersion1 := "TG1682_4.4s24_DEV_sey"
	req.Header.Set(common.HeaderFirmwareVersion, fwVersion1)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	bitmap, err := util.GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	assert.Equal(t, bitmap, rdoc.Bitmap)

	// call GET /supported_groups
	expectedEnabled := map[string]bool{
		"portforwarding":    true,
		"lan":               true,
		"wan":               true,
		"macbinding":        true,
		"hotspot":           false,
		"bridge":            false,
		"privatessid":       true,
		"homessid":          true,
		"radio":             false,
		"moca":              true,
		"xdns":              true,
		"advsecurity":       true,
		"mesh":              true,
		"aker":              true,
		"telemetry":         true,
		"statusreport":      false,
		"trafficreport":     false,
		"interfacereport":   false,
		"radioreport":       false,
		"telcovoip":         false,
		"telcovoice":        false,
		"wanmanager":        false,
		"voiceservice":      false,
		"wanfailover":       false,
		"cellularconfig":    false,
		"gwfailover":        false,
		"gwrestore":         false,
		"prioritizedmacs":   false,
		"connectedbuilding": false,
		"lldqoscontrol":     false,
	}

	// call GET /supported_groups to verify response
	req, err = http.NewRequest("GET", sgUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)

	var supportedGroupsGetResponse common.SupportedGroupsGetResponse
	err = json.Unmarshal(rbytes, &supportedGroupsGetResponse)
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedEnabled, supportedGroupsGetResponse.Data.Groups)

	// ==== step 2 add lan data ====
	subdocId := "lan"
	lanHexData := "81aa706172616d65746572739183a46e616d65b84465766963652e4448435076342e5365727665722e4c616ea576616c7565d99581a36c616e86b044686370536572766572456e61626c65c3ac4c616e495041646472657373a831302e302e302e31ad4c616e5375626e65744d61736bad3235352e3235352e3235352e30b2446863705374617274495041646472657373a831302e302e302e35b044686370456e64495041646472657373aa31302e302e302e323030a94c6561736554696d65d3000000000002a300a86461746154797065d3000000000000000c"

	lanBytes, err := hex.DecodeString(lanHexData)
	assert.NilError(t, err)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// ==== step 3 GET /config for fw version 1 =====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	rdkSupportedDocsHeaderStr = "16777231,33554435,50331649,67108865,83886081,100663297,117440513,134217729"
	req.Header.Set(common.HeaderSupportedDocs, rdkSupportedDocsHeaderStr)
	req.Header.Set(common.HeaderFirmwareVersion, fwVersion1)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)

	// ==== step 4 GET /supported_groups to verify the bitmaps ====
	req, err = http.NewRequest("GET", sgUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)

	err = json.Unmarshal(rbytes, &supportedGroupsGetResponse)
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedEnabled, supportedGroupsGetResponse.Data.Groups)

	// ==== step 5 setup supported docs for fw version 2 =====
	sids := strings.Split(rdkSupportedDocsHeaderStr, ",")

	newGroup1Bitarray := "00000001 0000 0000 0000 0000 0011 0011"
	group1Bitmap, err := util.BitarrayToBitmap(newGroup1Bitarray)
	assert.NilError(t, err)
	sids[0] = fmt.Sprintf("%v", group1Bitmap)
	expectedEnabled["wan"] = false
	expectedEnabled["macbinding"] = false
	expectedEnabled["hotspot"] = true
	expectedEnabled["bridge"] = true

	newGroup2Bitarray := "00000010 0000 0000 0000 0000 0000 0110"
	group2Bitmap, err := util.BitarrayToBitmap(newGroup2Bitarray)
	assert.NilError(t, err)
	sids[1] = fmt.Sprintf("%v", group2Bitmap)
	expectedEnabled["privatessid"] = false
	expectedEnabled["radio"] = true

	rdkSupportedDocsHeaderStr = strings.Join(sids, ",")

	// ==== step 6 GET /config with fw version2 and a diff supported docs header
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderSupportedDocs, rdkSupportedDocsHeaderStr)
	fwVersion2 := "TG1682_4.6s24_DEV_sey"
	req.Header.Set(common.HeaderFirmwareVersion, fwVersion2)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	mparts, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)

	// ==== step 7 GET /supported_groups to verify the bitmaps ====
	req, err = http.NewRequest("GET", sgUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)

	err = json.Unmarshal(rbytes, &supportedGroupsGetResponse)
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedEnabled, supportedGroupsGetResponse.Data.Groups)
}

func TestSupportedGroupsHandlerTelcovoice(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== step 1 test when no data ====
	cpeMac := util.GenerateRandomCpeMac()

	// call GET /supported_groups when no data
	sgUrl := fmt.Sprintf("/api/v1/device/%v/supported_groups", cpeMac)
	req, err := http.NewRequest("GET", sgUrl, nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// call GET /config to add supported-doc header
	configUrl := fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	rdkSupportedDocsHeaderStr := "16777231,33554433,67108865,83886081,100663297,117440513,134217729,184549378,201326595"
	req.Header.Set(common.HeaderSupportedDocs, rdkSupportedDocsHeaderStr)
	fwVersion1 := "TG1682_4.4s24_DEV_sey"
	req.Header.Set(common.HeaderFirmwareVersion, fwVersion1)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	bitmap, err := util.GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	assert.Equal(t, bitmap, rdoc.Bitmap)

	// call GET /supported_groups
	expectedEnabled := map[string]bool{
		"advsecurity":       true,
		"aker":              true,
		"bridge":            false,
		"cellularconfig":    false,
		"homessid":          false,
		"hotspot":           false,
		"interfacereport":   false,
		"lan":               true,
		"macbinding":        true,
		"mesh":              true,
		"moca":              false,
		"portforwarding":    true,
		"privatessid":       true,
		"radio":             false,
		"radioreport":       false,
		"statusreport":      false,
		"telcovoip":         false,
		"telcovoice":        true,
		"telemetry":         true,
		"trafficreport":     false,
		"voiceservice":      false,
		"wan":               true,
		"wanfailover":       true,
		"wanmanager":        true,
		"xdns":              true,
		"gwfailover":        false,
		"gwrestore":         false,
		"prioritizedmacs":   false,
		"connectedbuilding": false,
		"lldqoscontrol":     false,
	}

	// call GET /supported_groups to verify response
	req, err = http.NewRequest("GET", sgUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)

	var supportedGroupsGetResponse common.SupportedGroupsGetResponse
	err = json.Unmarshal(rbytes, &supportedGroupsGetResponse)
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedEnabled, supportedGroupsGetResponse.Data.Groups)
}

func TestSupportedGroupsHandlerWithPrioritizedmacsAndConnectedbuilding(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== step 1 test when no data ====
	cpeMac := util.GenerateRandomCpeMac()

	// call GET /supported_groups when no data
	sgUrl := fmt.Sprintf("/api/v1/device/%v/supported_groups", cpeMac)
	req, err := http.NewRequest("GET", sgUrl, nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// call GET /config to add supported-doc header
	configUrl := fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	rdkSupportedDocsHeaderStr := "16777311,33554435,50331649,67108865,83886081,100663297,117440513,134217729,201326594,218103809,251658241,268435457,285212673"
	req.Header.Set(common.HeaderSupportedDocs, rdkSupportedDocsHeaderStr)
	fwVersion1 := "TG1682_4.4s24_DEV_sey"
	req.Header.Set(common.HeaderFirmwareVersion, fwVersion1)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	bitmap, err := util.GetCpeBitmap(rdkSupportedDocsHeaderStr)
	assert.NilError(t, err)
	assert.Equal(t, bitmap, rdoc.Bitmap)

	// call GET /supported_groups
	expectedEnabled := map[string]bool{
		"advsecurity":       true,
		"aker":              true,
		"bridge":            false,
		"cellularconfig":    false,
		"homessid":          true,
		"hotspot":           true,
		"interfacereport":   false,
		"lan":               true,
		"macbinding":        true,
		"mesh":              true,
		"moca":              true,
		"portforwarding":    true,
		"privatessid":       true,
		"radio":             false,
		"radioreport":       false,
		"statusreport":      false,
		"telcovoip":         false,
		"telcovoice":        false,
		"telemetry":         true,
		"trafficreport":     false,
		"voiceservice":      true,
		"wan":               true,
		"wanfailover":       true,
		"wanmanager":        false,
		"xdns":              true,
		"gwfailover":        true,
		"gwrestore":         false,
		"prioritizedmacs":   true,
		"connectedbuilding": true,
		"lldqoscontrol":     true,
	}

	// call GET /supported_groups to verify response
	req, err = http.NewRequest("GET", sgUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)

	var supportedGroupsGetResponse common.SupportedGroupsGetResponse
	err = json.Unmarshal(rbytes, &supportedGroupsGetResponse)
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedEnabled, supportedGroupsGetResponse.Data.Groups)
}
