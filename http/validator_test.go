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

	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestValidatorDisabled(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := "foobar"
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
	rdoc, err := server.GetRootDocument(strings.ToUpper(cpeMac))
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// check the root version through API
	rootdocUrl := fmt.Sprintf("/api/v1/device/%v/rootdocument", cpeMac)
	req, err = http.NewRequest("GET", rootdocUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()

	var getResp HttGetRootDocumentResponse
	err = json.Unmarshal(rbytes, &getResp)
	assert.NilError(t, err)
	assert.Assert(t, len(getResp.Data.Version) > 0)

	// delete
	req, err = http.NewRequest("DELETE", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get but expect 404
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// check the root doc version
	rdoc, err = server.GetRootDocument(strings.ToUpper(cpeMac))
	assert.NilError(t, err)
	assert.Equal(t, rdoc.Version, "")

	// check the root version through API
	req, err = http.NewRequest("GET", rootdocUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()

	getResp = HttGetRootDocumentResponse{}
	err = json.Unmarshal(rbytes, &getResp)
	assert.NilError(t, err)
	assert.Assert(t, len(getResp.Data.Version) == 0)
}

func TestValidatorEnabled(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := "foobar"
	subdocId := "lan"

	lanHexData := "81aa706172616d65746572739183a46e616d65b84465766963652e4448435076342e5365727665722e4c616ea576616c7565d99581a36c616e86b044686370536572766572456e61626c65c3ac4c616e495041646472657373a831302e302e302e31ad4c616e5375626e65744d61736bad3235352e3235352e3235352e30b2446863705374617274495041646472657373a831302e302e302e35b044686370456e64495041646472657373aa31302e302e302e323030a94c6561736554696d65d3000000000002a300a86461746154797065d3000000000000000c"

	lanBytes, err := hex.DecodeString(lanHexData)
	assert.NilError(t, err)

	// post with mac check
	server.SetValidateMacEnabled(true)
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// post without mac check
	server.SetValidateMacEnabled(false)
	req, err = http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get with mac check
	server.SetValidateMacEnabled(true)
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// get without mac check
	server.SetValidateMacEnabled(false)
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)

	// check the root doc version
	rdoc, err := server.GetRootDocument(strings.ToUpper(cpeMac))
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)

	// check the root version through API with mac check
	server.SetValidateMacEnabled(true)
	rootdocUrl := fmt.Sprintf("/api/v1/device/%v/rootdocument", cpeMac)
	req, err = http.NewRequest("GET", rootdocUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// check the root version through API without mac check
	server.SetValidateMacEnabled(false)
	req, err = http.NewRequest("GET", rootdocUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()

	var getResp HttGetRootDocumentResponse
	err = json.Unmarshal(rbytes, &getResp)
	assert.NilError(t, err)
	assert.Assert(t, len(getResp.Data.Version) > 0)

	// delete with mac check
	server.SetValidateMacEnabled(true)
	req, err = http.NewRequest("DELETE", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)

	// delete without mac check
	server.SetValidateMacEnabled(false)
	req, err = http.NewRequest("DELETE", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get but expect 404
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// check the root doc version
	rdoc, err = server.GetRootDocument(strings.ToUpper(cpeMac))
	assert.NilError(t, err)
	assert.Equal(t, rdoc.Version, "")

	// check the root version through API
	req, err = http.NewRequest("GET", rootdocUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()

	getResp = HttGetRootDocumentResponse{}
	err = json.Unmarshal(rbytes, &getResp)
	assert.NilError(t, err)
	assert.Assert(t, len(getResp.Data.Version) == 0)
}

func TestValidatorWithLowerCase(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	server.SetValidateMacEnabled(true)

	cpeMac := util.GenerateRandomCpeMac()
	lowerCpeMac := strings.ToLower(cpeMac)

	// ==== step 1 setup lan subdoc ====
	// post
	subdocId := "lan"
	lanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", lowerCpeMac, subdocId)
	lanBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", lanUrl, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
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
	wanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", lowerCpeMac, subdocId)
	wanBytes := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", wanUrl, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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

	// ==== GET /config ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", lowerCpeMac)
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
	// etag := res.Header.Get(common.HeaderEtag)

	// parse the actual data
	mpart, ok := mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)

	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
}
