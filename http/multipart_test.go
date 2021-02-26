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
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"github.com/vmihailenco/msgpack/v4"
	"gotest.tools/assert"
)

func TestMultipartConfigHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true, nil)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()
	// ==== group 1 lan ====
	groupId := "lan"
	lanHexData := "81aa706172616d65746572739183a46e616d65b84465766963652e4448435076342e5365727665722e4c616ea576616c7565d99581a36c616e86b044686370536572766572456e61626c65c3ac4c616e495041646472657373a831302e302e302e31ad4c616e5375626e65744d61736bad3235352e3235352e3235352e30b2446863705374617274495041646472657373a831302e302e302e35b044686370456e64495041646472657373aa31302e302e302e323030a94c6561736554696d65d3000000000002a300a86461746154797065d3000000000000000c"

	lanBytes, err := hex.DecodeString(lanHexData)
	assert.NilError(t, err)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document?group_id=%v", cpeMac, groupId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
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

	// ==== group 2 wan ====
	groupId = "wan"
	wanHexData := "81aa706172616d65746572739183a46e616d65bf4465766963652e4e41542e585f434953434f5f434f4d5f444d5a2e44617461a576616c7565ba81a377616e82a6456e61626c65c2aa496e7465726e616c4950a0a86461746154797065d3000000000000000c"

	wanBytes, err := hex.DecodeString(wanHexData)
	assert.NilError(t, err)

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document?group_id=%v", cpeMac, groupId)
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

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.NilError(t, err)
	res.Body.Close()

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 2)
	etag := res.Header.Get("Etag")

	// parse the actual data
	mpart, ok := mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	var response common.TR181Output
	err = msgpack.Unmarshal(mpart.Bytes, &response)
	assert.NilError(t, err)
	parameters := response.Parameters
	assert.Equal(t, len(parameters), 1)
	lanMpartVersion := mpart.Version

	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
	err = msgpack.Unmarshal(mpart.Bytes, &response)
	assert.NilError(t, err)
	parameters = response.Parameters
	assert.Equal(t, len(parameters), 1)
	wanMpartVersion := mpart.Version
	_ = wanMpartVersion

	// ==== cal GET /config with if-none-match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set("If-None-Match", etag)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusNotModified)
	assert.NilError(t, err)
	res.Body.Close()

	// ==== cal GET /config with if-none-match partial match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,lan,wan", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	ifNoneMatch := fmt.Sprintf("foo,%v,bar", lanMpartVersion)
	req.Header.Set("If-None-Match", ifNoneMatch)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.NilError(t, err)
	res.Body.Close()

	mparts, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)

	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
	err = msgpack.Unmarshal(mpart.Bytes, &response)
	assert.NilError(t, err)
	parameters = response.Parameters
	assert.Equal(t, len(parameters), 1)
}

func TestCpeMiddleware(t *testing.T) {
	server := NewWebconfigServer(sc, true, nil)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()

	// add one sub doc
	groupId := "lan"
	lanHexData := "81aa706172616d65746572739183a46e616d65b84465766963652e4448435076342e5365727665722e4c616ea576616c7565d99581a36c616e86b044686370536572766572456e61626c65c3ac4c616e495041646472657373a831302e302e302e31ad4c616e5375626e65744d61736bad3235352e3235352e3235352e30b2446863705374617274495041646472657373a831302e302e302e35b044686370456e64495041646472657373aa31302e302e302e323030a94c6561736554696d65d3000000000002a300a86461746154797065d3000000000000000c"

	lanBytes, err := hex.DecodeString(lanHexData)
	assert.NilError(t, err)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document?group_id=%v", cpeMac, groupId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
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

	// ==== get /config on a new secure server ====
	server1 := NewWebconfigServer(sc, true, nil)
	router1 := server1.GetRouter(false)

	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router1).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusForbidden)
	assert.NilError(t, err)
	res.Body.Close()

	// get a token
	token := server1.Generate(cpeMac, 86400)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	res = ExecuteRequest(req, router1).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.NilError(t, err)
	res.Body.Close()
}

func TestFactoryReset(t *testing.T) {
	server := NewWebconfigServer(sc, true, nil)
	router := server.GetRouter(true)
	server.SetFactoryResetEnabled(true)

	cpeMac := util.GenerateRandomCpeMac()

	// ==== add group 1 lan ====
	groupId := "lan"
	lanHexData := "81aa706172616d65746572739183a46e616d65b84465766963652e4448435076342e5365727665722e4c616ea576616c7565d99581a36c616e86b044686370536572766572456e61626c65c3ac4c616e495041646472657373a831302e302e302e31ad4c616e5375626e65744d61736bad3235352e3235352e3235352e30b2446863705374617274495041646472657373a831302e302e302e35b044686370456e64495041646472657373aa31302e302e302e323030a94c6561736554696d65d3000000000002a300a86461746154797065d3000000000000000c"

	lanBytes, err := hex.DecodeString(lanHexData)
	assert.NilError(t, err)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document?group_id=%v", cpeMac, groupId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
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

	// ==== add group 2 wan ====
	groupId = "wan"
	wanHexData := "81aa706172616d65746572739183a46e616d65bf4465766963652e4e41542e585f434953434f5f434f4d5f444d5a2e44617461a576616c7565ba81a377616e82a6456e61626c65c2aa496e7465726e616c4950a0a86461746154797065d3000000000000000c"

	wanBytes, err := hex.DecodeString(wanHexData)
	assert.NilError(t, err)

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document?group_id=%v", cpeMac, groupId)
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

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.NilError(t, err)
	res.Body.Close()

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 2)

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

	// ==== cal GET /config with if-none-match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set("If-None-Match", "NONE")
	rdkSupportedDocsHeaderStr := "16777231,33554435,50331649,67108865,83886081,100663297,117440513,134217729"
	req.Header.Set(common.HeaderSupportedDocs, rdkSupportedDocsHeaderStr)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.NilError(t, err)
	res.Body.Close()

	// get /config again
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)
	assert.NilError(t, err)
	res.Body.Close()

	// get by group_id also return 404
	groupId = "lan"
	url = fmt.Sprintf("/api/v1/device/%v/document?group_id=%v", cpeMac, groupId)
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	groupId = "wan"
	url = fmt.Sprintf("/api/v1/device/%v/document?group_id=%v", cpeMac, groupId)
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// verify no data in root_version
	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, rdoc.Version(), "")
	assert.Assert(t, rdoc.Bitmap() > 0)
}
