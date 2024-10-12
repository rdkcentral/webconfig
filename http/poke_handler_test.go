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
	"strings"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

var (
	mockWebpaPokeResponse    = []byte(`{"parameters":[{"name":"Device.X_RDK_WebConfig.ForceSync","message":"Success"}],"statusCode":200}`)
	mockWebpaPoke403Response = []byte(`{"message": "Invalid partner_id", "statusCode": 403}`)
	mockWebpaPoke202Response = []byte(`{"parameters":[{"message":"Previous request is in progress","name":"Device.X_RDK_WebConfig.ForceSync"}],"statusCode":202}`)
)

func TestPokeHandler(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	// webpa mock server
	webpaMockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(mockWebpaPokeResponse)
		}))
	defer webpaMockServer.Close()
	server.SetWebpaHost(webpaMockServer.URL)
	targetWebpaHost := server.WebpaHost()
	assert.Equal(t, webpaMockServer.URL, targetWebpaHost)

	// ==== post new data ====
	lowerCpeMac := strings.ToLower(cpeMac)
	url := fmt.Sprintf("/api/v1/device/%v/poke", lowerCpeMac)
	req, err := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer foobar")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusNoContent)
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()

	// ==== poke telemetry expect 200 ====
	url = fmt.Sprintf("/api/v1/device/%v/poke?doc=telemetry", lowerCpeMac)
	req, err = http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer foobar")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
}

func TestPokeHandlerWithCpe(t *testing.T) {
	t.Skip()
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := "44AAF59D0F3A" // ok
	// cpeMac := "DCEB695C7812" // not found
	// cpeMac := "10868C6C5948" // expect 520

	// ==== post new data ====
	url := fmt.Sprintf("/api/v1/device/%v/poke", cpeMac)
	req, err := http.NewRequest("POST", url, nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
}

func TestBuildMqttSendDocument(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()

	// webpa mock server
	mockedMqttPokeResponse := []byte("Accepted\n")
	mqttMockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(mockedMqttPokeResponse)
		}))
	defer mqttMockServer.Close()
	server.SetMqttHost(mqttMockServer.URL)
	targetMqttHost := server.MqttHost()
	assert.Equal(t, mqttMockServer.URL, targetMqttHost)

	// ==== step 1 setup lan subdoc ====
	// post
	subdocId := "lan"
	lanUrl := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	lanBytes := util.RandomBytes(100, 150)
	req, err := http.NewRequest("POST", lanUrl, bytes.NewReader(lanBytes))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", lanUrl, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, lanBytes)
	state, err := strconv.Atoi(res.Header.Get("X-Subdocument-State"))
	assert.NilError(t, err)
	assert.Equal(t, state, common.PendingDownload)

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
	state, err = strconv.Atoi(res.Header.Get("X-Subdocument-State"))
	assert.NilError(t, err)
	assert.Equal(t, state, common.PendingDownload)

	// check the root doc version
	rdoc, err = server.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, len(rdoc.Version) > 0)
	etag2 := rdoc.Version

	assert.Assert(t, etag1 != etag2)

	// ==== step 3 check the length ===
	fields := make(log.Fields)
	document, err := db.BuildMqttSendDocument(server.DatabaseClient, cpeMac, fields)
	assert.NilError(t, err)
	assert.Equal(t, document.Length(), 2)

	// ==== step 4 send a document through mqtt ====
	mqttUrl := fmt.Sprintf("/api/v1/device/%v/poke?route=mqtt", cpeMac)
	req, err = http.NewRequest("POST", mqttUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusAccepted)

	// get to verify
	req, err = http.NewRequest("GET", lanUrl, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	state, err = strconv.Atoi(res.Header.Get("X-Subdocument-State"))
	assert.NilError(t, err)
	assert.Equal(t, state, common.InDeployment)

	// get to verify
	req, err = http.NewRequest("GET", wanUrl, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	state, err = strconv.Atoi(res.Header.Get("X-Subdocument-State"))
	assert.NilError(t, err)
	assert.Equal(t, state, common.InDeployment)

	// ==== step 6 check the length again ===
	fields = make(log.Fields)
	document, err = db.BuildMqttSendDocument(server.DatabaseClient, cpeMac, fields)
	assert.NilError(t, err)
	assert.Equal(t, document.Length(), 2)

	// ==== step 7 change the subdoc again ====
	lanBytes2 := util.RandomBytes(100, 150)
	req, err = http.NewRequest("POST", lanUrl, bytes.NewReader(lanBytes2))
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
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
	assert.DeepEqual(t, rbytes, lanBytes2)
	state, err = strconv.Atoi(res.Header.Get("X-Subdocument-State"))
	assert.NilError(t, err)
	assert.Equal(t, state, common.PendingDownload)

	// ==== step 8 check the document length ===
	document, err = db.BuildMqttSendDocument(server.DatabaseClient, cpeMac, fields)
	assert.NilError(t, err)
	assert.Equal(t, document.Length(), 2)

	// ==== step 9 send a document through mqtt ====
	req, err = http.NewRequest("POST", mqttUrl, nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusAccepted)

	// get to verify
	req, err = http.NewRequest("GET", lanUrl, nil)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	state, err = strconv.Atoi(res.Header.Get("X-Subdocument-State"))
	assert.NilError(t, err)
	assert.Equal(t, state, common.InDeployment)

	// ==== step 10 check the length again ===
	document, err = db.BuildMqttSendDocument(server.DatabaseClient, cpeMac, fields)
	assert.NilError(t, err)
	assert.Equal(t, document.Length(), 2)
}

func TestPokeHandlerInvalidMac(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac() + "foo"

	// webpa mock server
	webpaMockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(mockWebpaPokeResponse)
		}))
	defer webpaMockServer.Close()
	server.SetWebpaHost(webpaMockServer.URL)
	targetWebpaHost := server.WebpaHost()
	assert.Equal(t, webpaMockServer.URL, targetWebpaHost)

	// ==== post new data ====
	lowerCpeMac := strings.ToLower(cpeMac)
	url := fmt.Sprintf("/api/v1/device/%v/poke", lowerCpeMac)
	req, err := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer foobar")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
}

func TestPokeHandlerWebpa403(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	// webpa mock server
	webpaMockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write(mockWebpaPoke403Response)
		}))
	defer webpaMockServer.Close()
	server.SetWebpaHost(webpaMockServer.URL)
	targetWebpaHost := server.WebpaHost()
	assert.Equal(t, webpaMockServer.URL, targetWebpaHost)

	// ==== post new data ====
	lowerCpeMac := strings.ToLower(cpeMac)
	url := fmt.Sprintf("/api/v1/device/%v/poke?cpe_action=true", lowerCpeMac)
	req, err := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer foobar")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
}

func TestPokeHandlerWebpa202(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	// webpa mock server
	webpaMockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write(mockWebpaPoke202Response)
		}))
	defer webpaMockServer.Close()
	server.SetWebpaHost(webpaMockServer.URL)
	targetWebpaHost := server.WebpaHost()
	assert.Equal(t, webpaMockServer.URL, targetWebpaHost)

	// ==== post new data ====
	lowerCpeMac := strings.ToLower(cpeMac)
	url := fmt.Sprintf("/api/v1/device/%v/poke?cpe_action=true", lowerCpeMac)
	req, err := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer foobar")
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusAccepted)
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
}
