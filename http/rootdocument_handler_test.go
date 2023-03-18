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
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/vmihailenco/msgpack/v4"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

type HttGetRootDocumentResponse struct {
	Data    common.RootDocument `json:"data"`
	Message string              `json:"message"`
	Status  int                 `json:"status"`
}

func TestRootDocumentHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 GET /config device ====
	// boots up but with out data in db
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)

	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	supportedDocs1 := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729"
	firmwareVersion1 := "CGM4331COM_4.11p7s1_PROD_sey"
	modelName1 := "CGM4331COM"
	partner1 := "comcast"
	schemaVersion1 := "33554433-1.3,33554434-1.3"
	req.Header.Set(common.HeaderSupportedDocs, supportedDocs1)
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion1)
	req.Header.Set(common.HeaderModelName, modelName1)
	req.Header.Set(common.HeaderPartnerID, partner1)
	req.Header.Set(common.HeaderSchemaVersion, schemaVersion1)

	res := ExecuteRequest(req, router).Result()
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// read from db to compare version
	rootdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)

	expectedBitmap1, err := util.GetCpeBitmap(supportedDocs1)
	assert.NilError(t, err)
	expectedRootdoc := common.NewRootDocument(expectedBitmap1, firmwareVersion1, modelName1, partner1, schemaVersion1, "", "")
	assert.DeepEqual(t, rootdoc, expectedRootdoc)

	// ==== step 2 build lan subdoc ====
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
	rbytes, err := ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// ==== step 3 build wan subdoc ====
	subdocId = "wan"
	wanHexData := "81aa706172616d65746572739183a46e616d65bf4465766963652e4e41542e585f434953434f5f434f4d5f444d5a2e44617461a576616c7565ba81a377616e82a6456e61626c65c2aa496e7465726e616c4950a0a86461746154797065d3000000000000000c"

	wanBytes, err := hex.DecodeString(wanHexData)
	assert.NilError(t, err)

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== step 4 GET /config ====
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderSupportedDocs, supportedDocs1)
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion1)
	req.Header.Set(common.HeaderModelName, modelName1)
	req.Header.Set(common.HeaderPartnerID, partner1)
	req.Header.Set(common.HeaderSchemaVersion, schemaVersion1)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.NilError(t, err)
	res.Body.Close()

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 2)
	etag := res.Header.Get(common.HeaderEtag)
	assert.Assert(t, len(etag) > 0)

	// parse the actual data
	mpart, ok := mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	var response common.TR181Output
	err = msgpack.Unmarshal(mpart.Bytes, &response)
	assert.NilError(t, err)
	parameters := response.Parameters
	assert.Equal(t, len(parameters), 1)

	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
	err = msgpack.Unmarshal(mpart.Bytes, &response)
	assert.NilError(t, err)
	parameters = response.Parameters
	assert.Equal(t, len(parameters), 1)

	// ==== step 5 GET /rootdocument ====
	rootdocUrl := fmt.Sprintf("/api/v1/device/%v/rootdocument", cpeMac)
	req, err = http.NewRequest("GET", rootdocUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()

	var getResp HttGetRootDocumentResponse
	err = json.Unmarshal(rbytes, &getResp)
	assert.NilError(t, err)

	expectedRootdoc = common.NewRootDocument(expectedBitmap1, firmwareVersion1, modelName1, partner1, schemaVersion1, etag, "")
	assert.Equal(t, getResp.Data, *expectedRootdoc)
}

func TestPostRootDocumentHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 POST /rootdocument ====
	bitmap1 := 32479
	firmwareVersion1 := "CGM4331COM_4.11p7s1_PROD_sey"
	modelName1 := "CGM4331COM"
	partner1 := "comcast"
	schemaVersion1 := "33554433-1.3,33554434-1.3"
	etag := strconv.Itoa(int(time.Now().Unix()))
	queryParams1 := "stormReadyWifi=true&cellularMode=true"
	srcDoc1 := common.NewRootDocument(bitmap1, firmwareVersion1, modelName1, partner1, schemaVersion1, etag, queryParams1)
	bbytes, err := json.Marshal(srcDoc1)
	assert.NilError(t, err)

	rootdocUrl := fmt.Sprintf("/api/v1/device/%v/rootdocument", cpeMac)
	req, err := http.NewRequest("POST", rootdocUrl, bytes.NewReader(bbytes))
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== step 2 GET /rootdocument ====
	req, err = http.NewRequest("GET", rootdocUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	rbytes, err := ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()

	var getResp HttGetRootDocumentResponse
	err = json.Unmarshal(rbytes, &getResp)
	assert.NilError(t, err)

	assert.Equal(t, getResp.Data, *srcDoc1)
}
