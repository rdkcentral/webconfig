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
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestFilterOutputByBitmap(t *testing.T) {
	tsc1 := sc.Copy("webconfig.filter_output_by_bitmap_enabled=true")
	server := NewWebconfigServer(tsc1, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	now := time.Now()

	// ==== step 1 use epochNow as version and set a future expiry ====
	// post
	subdocId := "remotedebugger"
	remotedebuggerUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	remotedebuggerBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", remotedebuggerUrl, bytes.NewReader(remotedebuggerBytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)

	// prepare the version header
	reqHeaderVersion := strconv.Itoa(int(now.Unix()))
	req.Header.Set(common.HeaderSubdocumentVersion, reqHeaderVersion)

	// prepare a future expiry header
	futureT := now.AddDate(0, 0, 2)
	reqHeaderExpiry := strconv.Itoa(int(futureT.UnixNano() / 1000000))
	req.Header.Set(common.HeaderSubdocumentExpiry, reqHeaderExpiry)

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", remotedebuggerUrl, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()

	resHeaderVersion := res.Header.Get(common.HeaderSubdocumentVersion)
	assert.Equal(t, reqHeaderVersion, resHeaderVersion)
	resHeaderExpiry := res.Header.Get(common.HeaderSubdocumentExpiry)
	assert.Equal(t, reqHeaderExpiry, resHeaderExpiry)

	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, remotedebuggerBytes)

	// check the root doc version
	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// ==== step 2 get document ====
	supportedDocs1 := "16777217"
	firmwareVersion1 := "CGM4331COM_4.11p7s1_PROD_sey"
	modelName1 := "CGM4331COM"
	partner1 := "comcast"
	schemaVersion1 := "33554433-1.3,33554434-1.3"

	deviceConfigUrl := fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "0")
	req.Header.Set(common.HeaderSupportedDocs, supportedDocs1)
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion1)
	req.Header.Set(common.HeaderModelName, modelName1)
	req.Header.Set(common.HeaderPartnerID, partner1)
	req.Header.Set(common.HeaderSchemaVersion, schemaVersion1)

	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotFound)
}

func TestBitmapFilterExemptSubdocIds(t *testing.T) {
	tsc1 := sc.Copy(
		"webconfig.filter_output_by_bitmap_enabled=true",
		`webconfig.bitmap_filter_exempt_subdoc_ids=["remotedebugger"]`,
	)
	server := NewWebconfigServer(tsc1, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	now := time.Now()

	// ==== step 1 use epochNow as version and set a future expiry ====
	// post
	subdocId := "remotedebugger"
	remotedebuggerUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	remotedebuggerBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", remotedebuggerUrl, bytes.NewReader(remotedebuggerBytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)

	// prepare the version header
	reqHeaderVersion := strconv.Itoa(int(now.Unix()))
	req.Header.Set(common.HeaderSubdocumentVersion, reqHeaderVersion)

	// prepare a future expiry header
	futureT := now.AddDate(0, 0, 2)
	reqHeaderExpiry := strconv.Itoa(int(futureT.UnixNano() / 1000000))
	req.Header.Set(common.HeaderSubdocumentExpiry, reqHeaderExpiry)

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", remotedebuggerUrl, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()

	resHeaderVersion := res.Header.Get(common.HeaderSubdocumentVersion)
	assert.Equal(t, reqHeaderVersion, resHeaderVersion)
	resHeaderExpiry := res.Header.Get(common.HeaderSubdocumentExpiry)
	assert.Equal(t, reqHeaderExpiry, resHeaderExpiry)

	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, remotedebuggerBytes)

	// check the root doc version
	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// ==== step 2 get document ====
	supportedDocs1 := "16777217"
	firmwareVersion1 := "CGM4331COM_4.11p7s1_PROD_sey"
	modelName1 := "CGM4331COM"
	partner1 := "comcast"
	schemaVersion1 := "33554433-1.3,33554434-1.3"

	deviceConfigUrl := fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "0")
	req.Header.Set(common.HeaderSupportedDocs, supportedDocs1)
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion1)
	req.Header.Set(common.HeaderModelName, modelName1)
	req.Header.Set(common.HeaderPartnerID, partner1)
	req.Header.Set(common.HeaderSchemaVersion, schemaVersion1)

	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mpartMap, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mpartMap), 1)

	mpart, ok := mpartMap["remotedebugger"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, remotedebuggerBytes)
}
