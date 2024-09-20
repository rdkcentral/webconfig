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
	bbytes := util.RandomBytes(100, 150)

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
