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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestSubDocumentHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()
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
	rbytes, err := io.ReadAll(res.Body)
	_ = rbytes
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

	// check the root doc version
	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// delete
	req, err = http.NewRequest("DELETE", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get but expect 404
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// check the root doc version
	rdoc, err = server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, rdoc.Version, "")
}

func TestDeleteDocumentHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 setup lan subdoc ====
	// post
	subdocId := "lan"
	lanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	lanBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", lanUrl, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	_ = rbytes
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", lanUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// check the root doc version
	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)
	etag1 := rdoc.Version

	// ==== step 2 setup wan subdoc ====
	// post
	subdocId = "wan"
	wanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	wanBytes := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", wanUrl, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", wanUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, wanBytes)

	// check the root doc version
	rdoc, err = server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)
	etag2 := rdoc.Version

	assert.Assert(t, etag1 != etag2)

	// ==== step 3 call delete api to delete both subdocs ====
	url := fmt.Sprintf("/api/v1/device/%v/document", cpeMac)
	req, err = http.NewRequest("DELETE", url, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get to verify
	req, err = http.NewRequest("GET", lanUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// get to verify
	req, err = http.NewRequest("GET", wanUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	rootDocument, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, rootDocument.Version, "")
}

func TestPostWithDeviceId(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()

	deviceIds := []string{}
	for i := 0; i < 3; i++ {
		deviceIds = append(deviceIds, util.GenerateRandomCpeMac())
	}
	queryParams := strings.Join(deviceIds, ",")
	allMacs := []string{cpeMac}
	allMacs = append(allMacs, deviceIds...)

	// ==== step 1 setup lan subdoc ====
	// post
	subdocId := "lan"
	lanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v?device_id=%v", cpeMac, subdocId, queryParams)
	lanBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", lanUrl, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	_ = rbytes
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	for _, mac := range allMacs {
		url := fmt.Sprintf("/api/v1/device/%v/document/%v", mac, subdocId)
		req, err = http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/msgpack")
		assert.NilError(t, err)
		res = ExecuteRequest(req, router).Result()
		rbytes, err = io.ReadAll(res.Body)
		assert.NilError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.DeepEqual(t, rbytes, lanBytes)

		// check the root doc version
		rdoc, err := server.GetRootDocument(mac)
		assert.NilError(t, err)
		assert.Assert(t, len(rdoc.Version) > 0)
	}

	// ==== step 2 setup wan subdoc ====
	// post
	subdocId = "wan"
	wanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v?device_id=%v", cpeMac, subdocId, queryParams)
	wanBytes := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", wanUrl, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	for _, mac := range allMacs {
		url := fmt.Sprintf("/api/v1/device/%v/document/%v", mac, subdocId)
		req, err = http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/msgpack")
		assert.NilError(t, err)
		res = ExecuteRequest(req, router).Result()
		rbytes, err = io.ReadAll(res.Body)
		assert.NilError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.DeepEqual(t, rbytes, wanBytes)

		// check the root doc version
		rdoc, err := server.GetRootDocument(cpeMac)
		assert.NilError(t, err)
		assert.Assert(t, len(rdoc.Version) > 0)
	}
}

func TestSubDocumentHandlerWithVersionHeader(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 use epochNow as version ====
	// post
	subdocId := "gwrestore"
	gwrestoreUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	gwrestoreBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", gwrestoreUrl, bytes.NewReader(gwrestoreBytes))
	req.Header.Set("Content-Type", "application/msgpack")

	// prepare the version header
	now := time.Now()
	reqHeaderVersion := strconv.Itoa(int(now.Unix()))
	req.Header.Set(common.HeaderSubdocumentVersion, reqHeaderVersion)

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", gwrestoreUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	resHeaderVersion := res.Header.Get(common.HeaderSubdocumentVersion)
	assert.Equal(t, reqHeaderVersion, resHeaderVersion)

	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, gwrestoreBytes)

	// check the root doc version
	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// ==== step 2 use epochNow as version and set a future expiry ====
	// post
	subdocId = "remotedebugger"
	remotedebuggerUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	remotedebuggerBytes := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", remotedebuggerUrl, bytes.NewReader(remotedebuggerBytes))
	req.Header.Set("Content-Type", "application/msgpack")

	// prepare the version header
	reqHeaderVersion = strconv.Itoa(int(now.Unix()))
	req.Header.Set(common.HeaderSubdocumentVersion, reqHeaderVersion)

	// prepare a future expiry header
	futureT := now.AddDate(0, 0, 2)
	reqHeaderExpiry := strconv.Itoa(int(futureT.UnixNano() / 1000000))
	req.Header.Set(common.HeaderSubdocumentExpiry, reqHeaderExpiry)

	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", remotedebuggerUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()

	resHeaderVersion = res.Header.Get(common.HeaderSubdocumentVersion)
	assert.Equal(t, reqHeaderVersion, resHeaderVersion)
	resHeaderExpiry := res.Header.Get(common.HeaderSubdocumentExpiry)
	assert.Equal(t, reqHeaderExpiry, resHeaderExpiry)

	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, remotedebuggerBytes)

	// check the root doc version
	rdoc, err = server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// ==== step 3 add a regular subdoc ====
	// post
	subdocId = "lan"
	lanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	lanBytes := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", lanUrl, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", lanUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()

	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// check the root doc version
	rdoc, err = server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// ==== step 4 get document ====
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
	assert.Equal(t, len(mparts), 3)
	etag := res.Header.Get(common.HeaderEtag)

	subdocVersions := []string{etag}
	mpart, ok := mparts["gwrestore"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, gwrestoreBytes)
	subdocVersions = append(subdocVersions, mpart.Version)

	mpart, ok = mparts["remotedebugger"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, remotedebuggerBytes)
	subdocVersions = append(subdocVersions, mpart.Version)

	mpart, ok = mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	subdocVersions = append(subdocVersions, mpart.Version)

	// ==== step 5 get document again ====
	// ==== call GET /config with if-none-match ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,gwrestore,remotedebugger,lan", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	ifnonematch := strings.Join(subdocVersions, ",")
	req.Header.Set(common.HeaderIfNoneMatch, ifnonematch)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)
}

func TestSubDocumentHandlerWithExpiredVersionHeader(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 use epochNow as version ====
	// post
	subdocId := "gwrestore"
	gwrestoreUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	gwrestoreBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", gwrestoreUrl, bytes.NewReader(gwrestoreBytes))
	req.Header.Set("Content-Type", "application/msgpack")

	// prepare a version header
	now := time.Now()
	reqHeaderVersion := strconv.Itoa(int(now.Unix()))
	req.Header.Set(common.HeaderSubdocumentVersion, reqHeaderVersion)

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", gwrestoreUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	resHeaderVersion := res.Header.Get(common.HeaderSubdocumentVersion)
	assert.Equal(t, reqHeaderVersion, resHeaderVersion)

	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, gwrestoreBytes)

	// check the root doc version
	rdoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// ==== step 2 use epochNow as version and set a future expiry ====
	// post
	subdocId = "remotedebugger"
	remotedebuggerUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	remotedebuggerBytes := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", remotedebuggerUrl, bytes.NewReader(remotedebuggerBytes))
	req.Header.Set("Content-Type", "application/msgpack")

	// prepare the version header
	reqHeaderVersion = strconv.Itoa(int(now.Unix()))
	req.Header.Set(common.HeaderSubdocumentVersion, reqHeaderVersion)

	// prepare an past expiry header
	futureT := now.Add(time.Duration(-1) * time.Hour)
	reqHeaderExpiry := strconv.Itoa(int(futureT.UnixNano() / 1000000))
	req.Header.Set(common.HeaderSubdocumentExpiry, reqHeaderExpiry)

	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", remotedebuggerUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()

	resHeaderVersion = res.Header.Get(common.HeaderSubdocumentVersion)
	assert.Equal(t, reqHeaderVersion, resHeaderVersion)
	resHeaderExpiry := res.Header.Get(common.HeaderSubdocumentExpiry)
	assert.Equal(t, reqHeaderExpiry, resHeaderExpiry)

	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, remotedebuggerBytes)

	// check the root doc version
	rdoc, err = server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// ==== step 3 add a regular subdoc ====
	// post
	subdocId = "lan"
	lanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	lanBytes := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", lanUrl, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", lanUrl, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()

	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// check the root doc version
	rdoc, err = server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// ==== step 4 get document ====
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

	subdocVersions := []string{etag}
	mpart, ok := mparts["gwrestore"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, gwrestoreBytes)
	subdocVersions = append(subdocVersions, mpart.Version)

	mpart, ok = mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	subdocVersions = append(subdocVersions, mpart.Version)

	// ==== step 5 get document again ====
	configUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,gwrestore,lan", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	ifnonematch := strings.Join(subdocVersions, ",")
	req.Header.Set(common.HeaderIfNoneMatch, ifnonematch)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)
}

func TestBadHeaderExpiryHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 2 use epochNow as version and set a future expiry ====
	// post
	subdocId := "remotedebugger"
	remotedebuggerUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	remotedebuggerBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", remotedebuggerUrl, bytes.NewReader(remotedebuggerBytes))
	req.Header.Set("Content-Type", "application/msgpack")

	// manage version and expiry headers
	now := time.Now()
	reqHeaderVersion := strconv.Itoa(int(now.Unix()))
	req.Header.Set(common.HeaderSubdocumentVersion, reqHeaderVersion)
	futureT := now.AddDate(0, 0, 2)
	reqHeaderExpiry := strconv.Itoa(int(futureT.UnixNano()/1000000)) + "xxx"
	req.Header.Set(common.HeaderSubdocumentExpiry, reqHeaderExpiry)

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)
}
