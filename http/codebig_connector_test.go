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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/assert"
)

var (
	mockedCodebigResponse = []byte(`{"access_token":"one_mock_token","expires_in":86400,"scope":"scope1 scope2 scope3","token_type":"Bearer"}`)
)

func TestCodebigConnector(t *testing.T) {
	server := NewWebconfigServer(sc, true)

	// codebig mock server
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(mockedCodebigResponse)
		}))
	server.SetCodebigHost(mockServer.URL)
	targetCodebigHost := server.CodebigHost()
	assert.Equal(t, mockServer.URL, targetCodebigHost)

	// ==== post new data ====
	token, err := server.GetToken(nil)
	assert.NilError(t, err)
	assert.Equal(t, token, "one_mock_token")
}

func TestCodebigConnectorWithCpe(t *testing.T) {
	t.Skip()
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	// cpeMac := "44AAF59D0F3A" // ok
	// cpeMac := "DCEB695C7812" // not found
	cpeMac := "10868C6C5948" // expect 520

	// ==== post new data ====
	url := fmt.Sprintf("/api/v1/device/%v/poke", cpeMac)
	req, err := http.NewRequest("POST", url, nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	_, err = ioutil.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
}
