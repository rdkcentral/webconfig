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
	"io/ioutil"
	"net/http"
	"testing"

	"gotest.tools/assert"
)

func TestSimpleHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true, nil)
	router := server.GetRouter(true)

	// ==== test version api ====
	req, err := http.NewRequest("GET", "/api/v1/version", nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, 200)

	rbytes, err := ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	t.Log(string(rbytes))

	// ==== test monitor api ====
	req, err = http.NewRequest("GET", "/monitor", nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, 200)

	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, len(rbytes), 0)

	// ==== test monitor api by HEAD ====
	req, err = http.NewRequest("HEAD", "/monitor", nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, 200)

	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, len(rbytes), 0)

	// ==== test server config api ====
	req, err = http.NewRequest("GET", "/api/v1/config", nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, 200)

	rbytes, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	t.Log(string(rbytes))

	// get the expected config file
	configBytes, err := ioutil.ReadFile(testConfigFile)
	assert.NilError(t, err)
	assert.DeepEqual(t, rbytes, configBytes)
}
