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
	"testing"

	"github.com/google/uuid"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestRefSubDocumentHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	refId := uuid.New().String()
	bbytes := common.RandomBytes(100, 150)

	// post
	url := fmt.Sprintf("/api/v1/reference/%v/document", refId)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bbytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, bbytes)

	// delete
	req, err = http.NewRequest("DELETE", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get but expect 404
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)
}

func TestSubDocumentWithInvalidRefDoc(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	referenceIndicatorBytes := make([]byte, 4)

	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 setup refdoc1 and subdoc1 ====
	refId1 := uuid.New().String()
	bbytes1 := common.RandomBytes(100, 150)
	subdocId1 := "defaultrfc"

	// post
	url := fmt.Sprintf("/api/v1/reference/%v/document", refId1)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bbytes1))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, bbytes1)

	// link
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId1)
	xbytes := []byte(refId1)
	tmpbytes := append(referenceIndicatorBytes, xbytes...)
	req, err = http.NewRequest("POST", url, bytes.NewReader(tmpbytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== step 2 setup refdoc2 and subdoc2 ====
	refId2 := uuid.New().String()
	bbytes2 := common.RandomBytes(100, 150)
	subdocId2 := "defaulttelemetry"

	// post
	url = fmt.Sprintf("/api/v1/reference/%v/document", refId2)
	req, err = http.NewRequest("POST", url, bytes.NewReader(bbytes2))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, bbytes2)

	// link
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId2)
	xbytes = []byte(refId2)
	tmpbytes = append(referenceIndicatorBytes, xbytes...)
	req, err = http.NewRequest("POST", url, bytes.NewReader(tmpbytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== step 3 GET /config ====
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
	assert.Equal(t, len(mpartMap), 2)

	// parse the actual data
	mpart, ok := mpartMap[subdocId1]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, bbytes1)

	mpart, ok = mpartMap[subdocId2]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, bbytes2)

	// ==== step 4 setup 3rd subdoc but link it to an non-existent refdoc3 ====
	refId3 := uuid.New().String()
	subdocId3 := "defaultdcm"

	// link
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId3)
	xbytes = []byte(refId3)
	tmpbytes = append(referenceIndicatorBytes, xbytes...)
	req, err = http.NewRequest("POST", url, bytes.NewReader(tmpbytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== step 5 GET /config returns 2 subdocs and no errors ====
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mpartMap, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mpartMap), 2)

	// parse the actual data
	mpart, ok = mpartMap[subdocId1]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, bbytes1)

	mpart, ok = mpartMap[subdocId2]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, bbytes2)
}
