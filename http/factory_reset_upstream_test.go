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
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestFactoryResetWithoutUpstream(t *testing.T) {
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
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
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
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
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
	mpart, ok := mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)

	// ==== GET /config but with header changes without mock ====
	server.SetUpstreamEnabled(false)

	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	req.Header.Set(common.HeaderIfNoneMatch, "NONE")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// verify Document is now empty
	fields := make(log.Fields)
	_, err = server.GetDocument(cpeMac, fields)
	assert.Assert(t, server.IsDbNotFound(err))
}

func TestFactoryResetWithUpstream(t *testing.T) {
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
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
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
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
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
	mpart, ok := mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	// lanVersion := mpart.Version
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)

	// ==== setup mock upstream server ====
	fields := make(log.Fields)
	mockDoc, err := server.GetDocument(cpeMac, fields)
	assert.NilError(t, err)

	mockRootDoc, err := server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	mockDoc.SetRootDocument(mockRootDoc)

	mockDoc.DeleteSubDocument("wan")
	assert.NilError(t, err)

	mockBytes, err := mockDoc.Bytes()
	assert.NilError(t, err)

	db.RefreshRootDocumentVersion(mockDoc)
	refreshedRootVersion := mockDoc.RootVersion()

	upstreamMockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// build the response
			for k := range r.Header {
				w.Header().Set(k, r.Header.Get(k))
			}
			w.Header().Set(common.HeaderContentLength, strconv.Itoa(len(mockBytes)))
			ifNoneMatch := refreshedRootVersion
			w.Header().Set(common.HeaderEtag, ifNoneMatch)
			w.WriteHeader(http.StatusOK)
			w.Write(mockBytes)
		}))

	server.SetUpstreamHost(upstreamMockServer.URL)
	targetUpstreamHost := server.UpstreamHost()
	assert.Equal(t, upstreamMockServer.URL, targetUpstreamHost)
	defer upstreamMockServer.Close()

	// ==== GET /config but with header changes without mock ====
	server.SetUpstreamEnabled(true)

	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	req.Header.Set(common.HeaderIfNoneMatch, "NONE")
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
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	mpart, ok = mparts["wan"]
	assert.Assert(t, !ok)

	// verify Document is now empty
	doc, err := server.GetDocument(cpeMac, fields)
	assert.NilError(t, err)
	assert.Equal(t, doc.Length(), 1)
}
