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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db/cassandra"
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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

	// ==== call GET /config with if-none-match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, etag)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	rbytes, err = io.ReadAll(res.Body)
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

	// test root_document lock
	rootdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	rootdoc.LockedTill = int(time.Now().UnixMilli()) + 1000
	err = server.SetRootDocument(cpeMac, rootdoc)
	assert.NilError(t, err)

	// get document again without the feature flag enabled
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get document again with the feature flag enabled
	server.SetLockRootDocumentEnabled(true)

	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusConflict)

	time.Sleep(time.Duration(1) * time.Second)

	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)

	// get a token
	token := server1.Generate(cpeMac, 86400)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	res = ExecuteRequest(req, router1).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// change the min trust to 1000
	server1.SetMinTrust(1000)
	assert.Equal(t, 1000, server1.MinTrust())
	zeroToken := server1.Generate(cpeMac, 86400, 0)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", zeroToken))
	res = ExecuteRequest(req, router1).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)

	// change the min trust back to 0
	server1.SetMinTrust(0)
	assert.Equal(t, 0, server1.MinTrust())
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", zeroToken))
	res = ExecuteRequest(req, router1).Result()
	_, err = io.ReadAll(res.Body)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	rbytes, err = io.ReadAll(res.Body)
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
	rbytes, err = io.ReadAll(res.Body)
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
	contentType := respHeader.Get(common.HeaderContentType)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== GET /config ====
	deviceConfigUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusServiceUnavailable)

	// ==== GET /config reset schema version ====
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
			if rbytes, err := io.ReadAll(r.Body); err == nil {
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
	rbytes, err = io.ReadAll(res.Body)
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
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)

	// ==== test for partial match ====
	req, err = http.NewRequest("GET", qparamsDeviceConfigUrl, nil)
	req.Header.Set(common.HeaderSchemaVersion, "33554433-1.3,33554434-1.3")
	req.Header.Set(common.HeaderIfNoneMatch, mismatchedIfNoneMatch)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	contentType := respHeader.Get(common.HeaderContentType)
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
			if rbytes, err := io.ReadAll(r.Body); err == nil {
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
	contentType = respHeader.Get(common.HeaderContentType)
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
	contentType = respHeader.Get(common.HeaderContentType)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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

	// ==== call GET /config with if-none-match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,lan,wan", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	header1 := "NONE," + lanMpartVersion
	req.Header.Set(common.HeaderIfNoneMatch, header1)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== call GET /config with if-none-match ====
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	header2 := "NONE,123"
	req.Header.Set(common.HeaderIfNoneMatch, header2)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== call GET /config with if-none-match ====
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	header3 := etag + ",123"
	req.Header.Set(common.HeaderIfNoneMatch, header3)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)
}

func TestStateCorrectionEnabled(t *testing.T) {
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== group 3 mesh ====
	subdocId = "mesh"
	meshBytes := util.RandomBytes(m, n)
	assert.NilError(t, err)

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(meshBytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, meshBytes)

	// ==== group 3 moca ====
	subdocId = "moca"
	mocaBytes := util.RandomBytes(m, n)
	assert.NilError(t, err)

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(mocaBytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, mocaBytes)

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mpartMap, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mpartMap), 4)
	etag := res.Header.Get(common.HeaderEtag)

	// parse the actual data
	mpart, ok := mpartMap["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	lanVersion := mpart.Version

	mpart, ok = mpartMap["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
	wanVersion := mpart.Version

	mpart, ok = mpartMap["mesh"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, meshBytes)
	meshVersion := mpart.Version

	mpart, ok = mpartMap["moca"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, mocaBytes)
	mocaVersion := mpart.Version + "x"

	// verify all states are in-deployment
	lanSubdocument, err := server.GetSubDocument(cpeMac, "lan")
	assert.NilError(t, err)
	assert.Equal(t, lanSubdocument.GetState(), common.InDeployment)

	wanSubdocument, err := server.GetSubDocument(cpeMac, "wan")
	assert.NilError(t, err)
	assert.Equal(t, wanSubdocument.GetState(), common.InDeployment)

	meshSubdocument, err := server.GetSubDocument(cpeMac, "mesh")
	assert.NilError(t, err)
	assert.Equal(t, meshSubdocument.GetState(), common.InDeployment)

	mocaSubdocument, err := server.GetSubDocument(cpeMac, "moca")
	assert.NilError(t, err)
	assert.Equal(t, mocaSubdocument.GetState(), common.InDeployment)

	// ==== setup special error conditions to test state correction scenario ====
	lanState := common.PendingDownload
	lanSubdocument.SetState(&lanState)
	err = server.SetSubDocument(cpeMac, "lan", lanSubdocument)
	assert.NilError(t, err)

	wanState := common.InDeployment
	wanSubdocument.SetState(&wanState)
	err = server.SetSubDocument(cpeMac, "wan", wanSubdocument)
	assert.NilError(t, err)

	meshState := common.Failure
	meshErrorCode := 307
	meshErrorDetails := "NACK:OneWifi,"
	meshSubdocument.SetState(&meshState)
	meshSubdocument.SetErrorCode(&meshErrorCode)
	meshSubdocument.SetErrorDetails(&meshErrorDetails)
	err = server.SetSubDocument(cpeMac, "mesh", meshSubdocument)
	assert.NilError(t, err)

	mocaState := common.PendingDownload
	mocaSubdocument.SetState(&mocaState)
	err = server.SetSubDocument(cpeMac, "moca", mocaSubdocument)
	assert.NilError(t, err)

	// ==== call GET /config again with if-none-match and expect 304 ====
	server.SetStateCorrectionEnabled(false)

	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,lan,wan,mesh,moca", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	header1 := fmt.Sprintf("%v,%v,%v,%v,%v", etag, lanVersion, wanVersion, meshVersion, mocaVersion)
	req.Header.Set(common.HeaderIfNoneMatch, header1)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)

	// verify the states remain unchanged in the case of 304
	lanSubdocument, err = server.GetSubDocument(cpeMac, "lan")
	assert.NilError(t, err)
	assert.Equal(t, lanSubdocument.GetState(), common.PendingDownload)
	oldLanUpdatedTime := *lanSubdocument.UpdatedTime()

	wanSubdocument, err = server.GetSubDocument(cpeMac, "wan")
	assert.NilError(t, err)
	assert.Equal(t, wanSubdocument.GetState(), common.InDeployment)
	oldWanUpdatedTime := *wanSubdocument.UpdatedTime()

	meshSubdocument, err = server.GetSubDocument(cpeMac, "mesh")
	assert.NilError(t, err)
	assert.Equal(t, meshSubdocument.GetState(), common.Failure)
	assert.Equal(t, *meshSubdocument.ErrorCode(), meshErrorCode)
	assert.Equal(t, *meshSubdocument.ErrorDetails(), meshErrorDetails)
	oldMeshUpdatedTime := *meshSubdocument.UpdatedTime()

	mocaSubdocument, err = server.GetSubDocument(cpeMac, "moca")
	assert.NilError(t, err)
	assert.Equal(t, mocaSubdocument.GetState(), common.PendingDownload)
	oldMocaUpdatedTime := *mocaSubdocument.UpdatedTime()

	// ==== enable the state correction flag and call GET /config again with if-none-match and expect 304 ====
	server.SetStateCorrectionEnabled(true)
	defer func() {
		server.SetStateCorrectionEnabled(false)
	}()

	time.Sleep(time.Duration(100) * time.Millisecond)

	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, header1)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)

	// verify all version-matched states remain unchanged in the case of 304
	lanSubdocument, err = server.GetSubDocument(cpeMac, "lan")
	assert.NilError(t, err)
	assert.Equal(t, lanSubdocument.GetState(), common.Deployed)
	assert.Assert(t, *lanSubdocument.UpdatedTime() > oldLanUpdatedTime)

	wanSubdocument, err = server.GetSubDocument(cpeMac, "wan")
	assert.NilError(t, err)
	assert.Equal(t, wanSubdocument.GetState(), common.Deployed)
	assert.Assert(t, *wanSubdocument.UpdatedTime() > oldWanUpdatedTime)

	meshSubdocument, err = server.GetSubDocument(cpeMac, "mesh")
	assert.NilError(t, err)
	assert.Equal(t, meshSubdocument.GetState(), common.Deployed)
	assert.Equal(t, *meshSubdocument.ErrorCode(), 0)
	assert.Equal(t, *meshSubdocument.ErrorDetails(), "")
	assert.Assert(t, *meshSubdocument.UpdatedTime() > oldMeshUpdatedTime)

	mocaSubdocument, err = server.GetSubDocument(cpeMac, "moca")
	assert.NilError(t, err)
	assert.Equal(t, mocaSubdocument.GetState(), common.PendingDownload)
	assert.Assert(t, *mocaSubdocument.UpdatedTime() == oldMocaUpdatedTime)
}

func TestCorruptedEncryptedDocumentHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	tdbclient, ok := server.DatabaseClient.(*cassandra.CassandraClient)
	if !ok {
		t.Skip("Only test in cassandra env")
	}

	cpeMac := util.GenerateRandomCpeMac()
	encSubdocIds := []string{}
	tdbclient.SetEncryptedSubdocIds(encSubdocIds)
	readSubDocIds := tdbclient.EncryptedSubdocIds()
	assert.DeepEqual(t, encSubdocIds, readSubDocIds)
	assert.Assert(t, !tdbclient.IsEncryptedGroup("privatessid"))

	// ==== step 1 setup lan subdoc ====
	// post
	subdocId := "lan"
	lanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	lanBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", lanUrl, bytes.NewReader(lanBytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	_ = rbytes
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", lanUrl, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// ==== step 2 setup wan subdoc ====
	// post
	subdocId = "wan"
	wanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	wanBytes := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", wanUrl, bytes.NewReader(wanBytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", wanUrl, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== step 3 setup privatessid subdoc ====
	// post
	subdocId = "privatessid"
	privatessidUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	privatessidBytes := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", privatessidUrl, bytes.NewReader(privatessidBytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", privatessidUrl, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, privatessidBytes)

	// ==== step 4 read the document ====
	supportedDocs1 := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729"
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
	assert.Equal(t, len(mpartMap), 3)

	// parse the actual data
	mpart, ok := mpartMap["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)

	mpart, ok = mpartMap["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)

	mpart, ok = mpartMap["privatessid"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, privatessidBytes)

	// ==== step 5 set privatessid as an encrypted subdoc ====
	encSubdocIds = []string{"privatessid"}
	tdbclient.SetEncryptedSubdocIds(encSubdocIds)
	readSubDocIds = tdbclient.EncryptedSubdocIds()
	assert.DeepEqual(t, encSubdocIds, readSubDocIds)
	assert.Assert(t, tdbclient.IsEncryptedGroup("privatessid"))

	// ==== step 6 read the document expect no error but 1 less subdoc ====
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

	mpartMap, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mpartMap), 2)

	// parse the actual data
	mpart, ok = mpartMap["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)

	mpart, ok = mpartMap["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)

	_, ok = mpartMap["privatessid"]
	assert.Assert(t, !ok)
}

func TestValidateQueryParams(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	server.SetQueryParamsValidationEnabled(true)
	assert.Assert(t, server.QueryParamsValidationEnabled())
	defer server.SetQueryParamsValidationEnabled(false)

	cpeMac := util.GenerateRandomCpeMac()
	// ==== group 1 lan ====
	subdocId := "lan"
	m, n := 50, 100
	lanBytes := util.RandomBytes(m, n)

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// case 1
	deviceConfigUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// case 2
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?foo=bar", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// case 3
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// case 4
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// case 5
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,foo", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "123,234")
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// case 6
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,privatessid,foo", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "123,234,345")
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// case 7
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,privatessid,homessid", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "123,234,345,456")
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// case 8
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "123")
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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

	// case 9 versions matched 304
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,lan,wan", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, matchedIfNoneMatch)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)
}
