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
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v4"
	"gotest.tools/assert"
)

func TestMultipartConfigHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()
	// ==== group 1 lan ====
	subdocId := "lan"
	lanHexData := "81aa706172616d65746572739183a46e616d65b84465766963652e4448435076342e5365727665722e4c616ea576616c7565d99581a36c616e86b044686370536572766572456e61626c65c3ac4c616e495041646472657373a831302e302e302e31ad4c616e5375626e65744d61736bad3235352e3235352e3235352e30b2446863705374617274495041646472657373a831302e302e302e35b044686370456e64495041646472657373aa31302e302e302e323030a94c6561736554696d65d3000000000002a300a86461746154797065d3000000000000000c"

	lanBytes, err := hex.DecodeString(lanHexData)
	assert.NilError(t, err)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// ==== group 2 wan ====
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
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 2)
	etag := res.Header.Get(common.HeaderEtag)

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
	req.Header.Set(common.HeaderIfNoneMatch, etag)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)

	// ==== cal GET /config with if-none-match partial match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,lan,wan", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	ifNoneMatch := fmt.Sprintf("foo,%v,bar", lanMpartVersion)
	req.Header.Set(common.HeaderIfNoneMatch, ifNoneMatch)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

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
	sc, err := common.GetTestServerConfig()
	if err != nil {
		panic(err)
	}
	if !sc.GetBoolean("webconfig.jwt.enabled") {
		t.Skip("webconfig.jwt.enabled = false")
	}

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()

	// add one sub doc
	subdocId := "lan"
	lanHexData := "81aa706172616d65746572739183a46e616d65b84465766963652e4448435076342e5365727665722e4c616ea576616c7565d99581a36c616e86b044686370536572766572456e61626c65c3ac4c616e495041646472657373a831302e302e302e31ad4c616e5375626e65744d61736bad3235352e3235352e3235352e30b2446863705374617274495041646472657373a831302e302e302e35b044686370456e64495041646472657373aa31302e302e302e323030a94c6561736554696d65d3000000000002a300a86461746154797065d3000000000000000c"

	lanBytes, err := hex.DecodeString(lanHexData)
	assert.NilError(t, err)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// ==== get /config on a new secure server ====
	server1 := NewWebconfigServer(sc, true)
	router1 := server1.GetRouter(false)

	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router1).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)

	// get a token
	token := server1.Generate(cpeMac, 86400)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	res = ExecuteRequest(req, router1).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
}

func TestVersionFiltering(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()
	// ==== group 1 lan ====
	subdocId := "lan"
	m, n := 50, 100
	lanBytes := util.RandomBytes(m, n)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// ==== group 2 wan ====
	subdocId = "wan"
	wanBytes := util.RandomBytes(m, n)

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 2)
	etag := res.Header.Get(common.HeaderEtag)
	mpart, ok := mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	lanMpartVersion := mpart.Version
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
	wanMpartVersion := mpart.Version
	matchedIfNoneMatch := fmt.Sprintf("%v,%v,%v", etag, lanMpartVersion, wanMpartVersion)

	// ==== call GET /config with if-none-match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, etag)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)

	// ==== call GET /config with if-none-match partial match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,lan,wan", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	ifNoneMatch := fmt.Sprintf("foo,%v,bar", lanMpartVersion)
	req.Header.Set(common.HeaderIfNoneMatch, ifNoneMatch)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)

	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)

	// ==== get mqtt/kafka response ====
	kHeader := make(http.Header)
	kHeader.Set(common.HeaderIfNoneMatch, ifNoneMatch)
	kHeader.Set(common.HeaderDocName, "root,lan,wan")
	kHeader.Set(common.HeaderDeviceId, cpeMac)
	kHeader.Set(common.HeaderSchemaVersion, "none")
	fields := make(log.Fields)
	status, respHeader, respBytes, err := BuildWebconfigResponse(server, kHeader, common.RouteMqtt, fields)
	assert.NilError(t, err)
	assert.Equal(t, status, http.StatusOK)
	contentType := respHeader.Get("Content-Type")
	assert.Assert(t, strings.Contains(contentType, "multipart/mixed"))
	mparts, err = util.ParseMultipart(respHeader, respBytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)

	// ==== get mqtt/kafka response and expect 304 ====
	kHeader = make(http.Header)
	kHeader.Set(common.HeaderIfNoneMatch, matchedIfNoneMatch)
	kHeader.Set(common.HeaderDocName, "root,lan,wan")
	kHeader.Set(common.HeaderDeviceId, cpeMac)
	kHeader.Set(common.HeaderSchemaVersion, "none")
	status, respHeader, respBytes, err = BuildWebconfigResponse(server, kHeader, common.RouteMqtt, fields)
	assert.NilError(t, err)
	assert.Equal(t, status, http.StatusNotModified)
	assert.Equal(t, len(respBytes), 0)
}

func TestUpstreamVersionFiltering(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()
	// ==== group 1 lan ====
	subdocId := "lan"
	m, n := 50, 100
	lanBytes := util.RandomBytes(m, n)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// ==== group 2 wan ====
	subdocId = "wan"
	wanBytes := util.RandomBytes(m, n)

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== GET /config ====
	deviceConfigUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 2)
	etag := res.Header.Get(common.HeaderEtag)
	mpart, ok := mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	lanMpartVersion := mpart.Version
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
	wanMpartVersion := mpart.Version
	matchedIfNoneMatch := fmt.Sprintf("%v,%v,%v", etag, lanMpartVersion, wanMpartVersion)
	mismatchedIfNoneMatch := fmt.Sprintf("foo,%v,bar", lanMpartVersion)

	// ==== GET /config but with header changes without mock ====
	server.SetUpstreamEnabled(true)
	server.SetUpstreamHost("http://localhost:12345")

	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	req.Header.Set(common.HeaderSchemaVersion, "33554433-1.3,33554434-1.3")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusServiceUnavailable)

	// ==== GET /config reset schema version ====
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusServiceUnavailable)

	// ==== GET /config but with header changes with mock ====
	upstreamMockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// build the response
			for k := range r.Header {
				w.Header().Set(k, r.Header.Get(k))
			}
			w.WriteHeader(http.StatusOK)
			if rbytes, err := ioutil.ReadAll(r.Body); err == nil {
				_, err := w.Write(rbytes)
				assert.NilError(t, err)
			}
		}))
	server.SetUpstreamHost(upstreamMockServer.URL)
	targetUpstreamHost := server.UpstreamHost()
	assert.Equal(t, upstreamMockServer.URL, targetUpstreamHost)
	defer upstreamMockServer.Close()

	// test again
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	req.Header.Set(common.HeaderSchemaVersion, "33554433-1.3,33554434-1.3")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	mparts, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 2)
	etag = res.Header.Get(common.HeaderEtag)
	mpart, ok = mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)

	// ==== test for 304 ====
	// need to NOT setting HeaderSchemaVersion to trigger the upstream
	qparamsDeviceConfigUrl := fmt.Sprintf("/api/v1/device/%v/config?group_id=root,lan,wan", cpeMac)
	req, err = http.NewRequest("GET", qparamsDeviceConfigUrl, nil)
	req.Header.Set(common.HeaderIfNoneMatch, matchedIfNoneMatch)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)

	// ==== test for partial match ====
	req, err = http.NewRequest("GET", qparamsDeviceConfigUrl, nil)
	req.Header.Set(common.HeaderSchemaVersion, "33554433-1.3,33554434-1.3")
	req.Header.Set(common.HeaderIfNoneMatch, mismatchedIfNoneMatch)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	mparts, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok = mparts["lan"]
	assert.Assert(t, !ok)
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
}

func TestMqttUpstreamVersionFiltering(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()
	// ==== group 1 lan ====
	subdocId := "lan"
	m, n := 50, 100
	lanBytes := util.RandomBytes(m, n)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// ==== group 2 wan ====
	subdocId = "wan"
	wanBytes := util.RandomBytes(m, n)

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== GET /config ====
	// ==== get mqtt/kafka response ====
	kHeader := make(http.Header)
	kHeader.Set(common.HeaderDeviceId, cpeMac)
	fields := make(log.Fields)
	status, respHeader, respBytes, err := BuildWebconfigResponse(server, kHeader, common.RouteMqtt, fields)
	assert.NilError(t, err)
	assert.Equal(t, status, http.StatusOK)
	contentType := respHeader.Get("Content-Type")
	assert.Assert(t, strings.Contains(contentType, "multipart/mixed"))
	mparts, err := util.ParseMultipart(respHeader, respBytes)
	assert.NilError(t, err)

	assert.Equal(t, len(mparts), 2)
	etag := res.Header.Get(common.HeaderEtag)
	mpart, ok := mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	lanMpartVersion := mpart.Version
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
	wanMpartVersion := mpart.Version
	matchedIfNoneMatch := fmt.Sprintf("%v,%v,%v", etag, lanMpartVersion, wanMpartVersion)
	mismatchedIfNoneMatch := fmt.Sprintf("foo,%v,bar", lanMpartVersion)

	// ==== GET /config but with header changes without mock ====
	server.SetUpstreamEnabled(true)
	server.SetUpstreamHost("http://localhost:12345")

	// ==== get mqtt/kafka response ====
	kHeader = make(http.Header)
	kHeader.Set(common.HeaderDeviceId, cpeMac)
	kHeader.Set(common.HeaderSchemaVersion, "33554433-1.3,33554434-1.3")
	fields = make(log.Fields)
	status, respHeader, respBytes, err = BuildWebconfigResponse(server, kHeader, common.RouteMqtt, fields)
	assert.Assert(t, err != nil)
	assert.Equal(t, status, http.StatusServiceUnavailable)

	// ==== GET /config reset schema version ====
	kHeader = make(http.Header)
	kHeader.Set(common.HeaderDeviceId, cpeMac)
	fields = make(log.Fields)
	status, respHeader, respBytes, err = BuildWebconfigResponse(server, kHeader, common.RouteMqtt, fields)
	assert.Assert(t, err != nil)
	assert.Equal(t, status, http.StatusServiceUnavailable)

	// ==== GET /config but with header changes with mock ====
	upstreamMockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// build the response
			for k := range r.Header {
				w.Header().Set(k, r.Header.Get(k))
			}
			w.WriteHeader(http.StatusOK)
			if rbytes, err := ioutil.ReadAll(r.Body); err == nil {
				_, err := w.Write(rbytes)
				assert.NilError(t, err)
			}
		}))
	server.SetUpstreamHost(upstreamMockServer.URL)
	targetUpstreamHost := server.UpstreamHost()
	assert.Equal(t, upstreamMockServer.URL, targetUpstreamHost)
	defer upstreamMockServer.Close()

	kHeader = make(http.Header)
	kHeader.Set(common.HeaderDeviceId, cpeMac)
	kHeader.Set(common.HeaderSchemaVersion, "33554433-1.3,33554434-1.3")
	fields = make(log.Fields)
	status, respHeader, respBytes, err = BuildWebconfigResponse(server, kHeader, common.RouteMqtt, fields)
	assert.NilError(t, err)
	assert.Equal(t, status, http.StatusOK)
	contentType = respHeader.Get("Content-Type")
	assert.Assert(t, strings.Contains(contentType, "multipart/mixed"))
	mparts, err = util.ParseMultipart(respHeader, respBytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 2)
	etag = res.Header.Get(common.HeaderEtag)
	mpart, ok = mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)

	// ==== test for 304 ====
	// need to NOT setting HeaderSchemaVersion to trigger the upstream
	kHeader = make(http.Header)
	kHeader.Set(common.HeaderIfNoneMatch, matchedIfNoneMatch)
	kHeader.Set(common.HeaderDocName, "root,lan,wan")
	kHeader.Set(common.HeaderDeviceId, cpeMac)
	kHeader.Set(common.HeaderSchemaVersion, "none")
	fields = make(log.Fields)
	status, respHeader, respBytes, err = BuildWebconfigResponse(server, kHeader, common.RouteMqtt, fields)
	assert.NilError(t, err)
	assert.Equal(t, status, http.StatusNotModified)

	// ==== test for partial match ====
	kHeader = make(http.Header)
	kHeader.Set(common.HeaderIfNoneMatch, mismatchedIfNoneMatch)
	kHeader.Set(common.HeaderDocName, "root,lan,wan")
	kHeader.Set(common.HeaderDeviceId, cpeMac)
	kHeader.Set(common.HeaderSchemaVersion, "33554433-1.3,33554434-1.3")
	fields = make(log.Fields)
	status, respHeader, respBytes, err = BuildWebconfigResponse(server, kHeader, common.RouteMqtt, fields)
	assert.NilError(t, err)
	assert.Equal(t, status, http.StatusOK)
	contentType = respHeader.Get("Content-Type")
	assert.Assert(t, strings.Contains(contentType, "multipart/mixed"))
	mparts, err = util.ParseMultipart(respHeader, respBytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok = mparts["lan"]
	assert.Assert(t, !ok)
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
}

func TestMultipartConfigMismatch(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	// ==== group 1 lan ====
	subdocId := "lan"
	m, n := 50, 100
	lanBytes := util.RandomBytes(m, n)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// ==== group 2 wan ====
	subdocId = "wan"
	wanBytes := util.RandomBytes(m, n)
	assert.NilError(t, err)

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 2)
	etag := res.Header.Get(common.HeaderEtag)

	// parse the actual data
	mpart, ok := mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	lanMpartVersion := mpart.Version

	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
	wanMpartVersion := mpart.Version
	_ = wanMpartVersion

	// ==== cal GET /config with if-none-match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,lan,wan", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	header1 := "NONE," + lanMpartVersion
	req.Header.Set(common.HeaderIfNoneMatch, header1)
	res = ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== cal GET /config with if-none-match ====
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	header2 := "NONE,123"
	req.Header.Set(common.HeaderIfNoneMatch, header2)
	res = ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== cal GET /config with if-none-match ====
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	header3 := etag + ",123"
	req.Header.Set(common.HeaderIfNoneMatch, header3)
	res = ExecuteRequest(req, router).Result()
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)
}
