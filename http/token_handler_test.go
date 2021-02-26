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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestTokenHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true, nil)
	server.SetTokenApiEnabled(false)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()

	// ==== post new data ====
	url := "/api/v1/token"
	sourceData := util.Dict{
		"mac": cpeMac,
		"ttl": 86400,
	}

	bbytes, err := json.Marshal(sourceData)
	assert.NilError(t, err)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bbytes))
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// ==== retry after the token api enabled ====
	server.SetTokenApiEnabled(true)
	router = server.GetRouter(true)

	req, err = http.NewRequest("POST", url, bytes.NewReader(bbytes))
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	rbytes, err := ioutil.ReadAll(res.Body)
	tpres := common.PostTokenResponse{}
	err = json.Unmarshal(rbytes, &tpres)
	assert.NilError(t, err)
	assert.Assert(t, len(tpres.Data) > 0)
}
