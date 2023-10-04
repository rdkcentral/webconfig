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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestUpstream(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.SetUpstreamEnabled(true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	m, n := 50, 100
	lanBytes := util.RandomBytes(m, n)
	wanBytes := util.RandomBytes(m, n)
	privatessidV13Bytes := util.RandomBytes(m, n)
	privatessidV14Bytes := util.RandomBytes(m, n)
	srcbytesMap := map[string][]byte{
		"privatessid": privatessidV13Bytes,
		"lan":         lanBytes,
		"wan":         wanBytes,
	}

	// ==== step 1 set up upstream mock server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// parse request
			reqBytes, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			// not necessarily always the case but we return 404 if the input is empty
			if len(reqBytes) == 0 {
				w.Header().Set("Content-type", common.MultipartContentType)
				w.Header().Set(common.HeaderEtag, "")
				w.Header().Set(common.HeaderStoreUpstreamResponse, "true")
				w.WriteHeader(http.StatusNotFound)
				return
			}

			mparts, err := util.ParseMultipartAsList(r.Header, reqBytes)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			// modify the payload
			newMparts := []common.Multipart{}
			for _, mpart := range mparts {
				if mpart.Name == "privatessid" {
					version := util.GetMurmur3Hash(privatessidV14Bytes)
					newMpart := common.Multipart{
						Name:    mpart.Name,
						Version: version,
						Bytes:   privatessidV14Bytes,
					}
					newMparts = append(newMparts, newMpart)
				} else {
					newMparts = append(newMparts, mpart)
				}
			}
			newRootVersion := db.HashRootVersion(newMparts)

			respBytes, err := common.WriteMultipartBytes(newMparts)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			// generate response
			w.Header().Set("Content-type", common.MultipartContentType)
			w.Header().Set(common.HeaderEtag, newRootVersion)
			w.Header().Set(common.HeaderStoreUpstreamResponse, "true")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(respBytes)
		}))
	server.SetUpstreamHost(mockServer.URL)
	targetUpstreamHost := server.UpstreamHost()
	assert.Equal(t, mockServer.URL, targetUpstreamHost)

	// ==== step 2 GET /config to create root document meta ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)

	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	supportedDocs1 := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729"
	firmwareVersion1 := "CGM4331COM_4.11p7s1_PROD_sey"
	modelName1 := "CGM4331COM"
	partner1 := "comcast"
	schemaVersion1 := "33554433-1.3,33554434-1.3"
	req.Header.Set(common.HeaderSupportedDocs, supportedDocs1)
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion1)
	req.Header.Set(common.HeaderModelName, modelName1)
	req.Header.Set(common.HeaderPartnerID, partner1)
	req.Header.Set(common.HeaderSchemaVersion, schemaVersion1)

	res := ExecuteRequest(req, router).Result()
	assert.NilError(t, err)
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// ==== step 3 add group privatessid ====
	subdocId := "privatessid"

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(privatessidV13Bytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, privatessidV13Bytes)

	// ==== step 4 add group lan ====
	subdocId = "lan"

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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

	// ==== step 5 add group wan ====
	subdocId = "wan"

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== step 6 GET /config ====
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	req.Header.Set(common.HeaderSupportedDocs, supportedDocs1)
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion1)
	req.Header.Set(common.HeaderModelName, modelName1)
	req.Header.Set(common.HeaderPartnerID, partner1)
	req.Header.Set(common.HeaderSchemaVersion, schemaVersion1)

	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.NilError(t, err)
	res.Body.Close()
	etag := res.Header.Get(common.HeaderEtag)
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 3)
	mpart, ok := mparts["privatessid"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, privatessidV13Bytes)
	privatessidVersion := mpart.Version

	mpart, ok = mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	lanVersion := mpart.Version

	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)
	wanVersion := mpart.Version

	// ==== step 7 verify the states are in_deployment ====
	subdocIds := []string{"privatessid", "lan", "wan"}
	for _, subdocId := range subdocIds {
		url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
		req, err = http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/msgpack")
		assert.NilError(t, err)
		res = ExecuteRequest(req, router).Result()
		rbytes, err = io.ReadAll(res.Body)
		assert.NilError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.DeepEqual(t, rbytes, srcbytesMap[subdocId])
		state, err := strconv.Atoi(res.Header.Get(common.HeaderSubdocumentState))
		assert.NilError(t, err)
		assert.Equal(t, state, common.InDeployment)
	}

	// ==== step 8 update the states ====
	for _, subdocId := range subdocIds {
		notifBody := fmt.Sprintf(`{"namespace": "%v", "application_status": "success", "updated_time": 1635976420, "cpe_doc_version": "984628970", "transaction_uuid": "6ef948f6-cbfa-4620-bde7-8acca1f95ba3_____005CFE970DE53C1"}`, subdocId)
		var m common.EventMessage
		err := json.Unmarshal([]byte(notifBody), &m)
		assert.NilError(t, err)
		fields := make(log.Fields)
		err = db.UpdateDocumentState(server.DatabaseClient, cpeMac, &m, fields)
		assert.NilError(t, err)
	}

	// ==== step 9 verify all states deployed ====
	for _, subdocId := range subdocIds {
		url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
		req, err = http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/msgpack")
		assert.NilError(t, err)
		res = ExecuteRequest(req, router).Result()
		rbytes, err = io.ReadAll(res.Body)
		assert.NilError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.DeepEqual(t, rbytes, srcbytesMap[subdocId])
		state, err := strconv.Atoi(res.Header.Get(common.HeaderSubdocumentState))
		assert.NilError(t, err)
		assert.Equal(t, state, common.Deployed)
	}

	// ==== step 10 GET /config with schemaVersion change to trigger upstream ====
	configUrl = configUrl + "?group_id=root,privatessid,lan,wan"
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	ifNoneMatch := fmt.Sprintf("%v,%v,%v,%v", etag, privatessidVersion, lanVersion, wanVersion)
	req.Header.Set(common.HeaderIfNoneMatch, ifNoneMatch)
	schemaVersion2 := "33554433-1.4,33554434-1.4"
	req.Header.Set(common.HeaderSupportedDocs, supportedDocs1)
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion1)
	req.Header.Set(common.HeaderModelName, modelName1)
	req.Header.Set(common.HeaderPartnerID, partner1)
	req.Header.Set(common.HeaderSchemaVersion, schemaVersion2)

	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.NilError(t, err)
	res.Body.Close()
	mparts, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)

	// ==== step 11 verify all states deployed ====
	// srcbytesMap changed
	srcbytesMap["privatessid"] = privatessidV14Bytes
	expectedStateMap := map[string]int{
		"privatessid": common.InDeployment,
		"lan":         common.Deployed,
		"wan":         common.Deployed,
	}
	for _, subdocId := range subdocIds {
		url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
		req, err = http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/msgpack")
		assert.NilError(t, err)
		res = ExecuteRequest(req, router).Result()
		rbytes, err = io.ReadAll(res.Body)
		assert.NilError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.DeepEqual(t, rbytes, srcbytesMap[subdocId])
		state, err := strconv.Atoi(res.Header.Get(common.HeaderSubdocumentState))
		assert.NilError(t, err)
		assert.Equal(t, state, expectedStateMap[subdocId])
	}
}

func TestUpstreamStateChangeFirmwareChange(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.SetUpstreamEnabled(true)
	router := server.GetRouter(true)
	cpeMac := util.GenerateRandomCpeMac()

	m, n := 50, 100
	lanBytes := util.RandomBytes(m, n)
	wanBytes := util.RandomBytes(m, n)
	privatessidBytes := util.RandomBytes(m, n)
	srcbytesMap := map[string][]byte{
		"privatessid": privatessidBytes,
		"lan":         lanBytes,
		"wan":         wanBytes,
	}

	// ==== step 1 set up upstream mock server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// parse request
			reqBytes, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			// not necessarily always the case but we return 404 if the input is empty
			if len(reqBytes) == 0 {
				w.Header().Set("Content-type", common.MultipartContentType)
				w.Header().Set(common.HeaderEtag, "")
				w.Header().Set(common.HeaderStoreUpstreamResponse, "true")
				w.WriteHeader(http.StatusNotFound)
				return
			}

			etag := r.Header.Get(common.HeaderEtag)
			respBytes := reqBytes

			// generate response
			w.Header().Set("Content-type", common.MultipartContentType)
			w.Header().Set(common.HeaderEtag, etag)
			w.Header().Set(common.HeaderStoreUpstreamResponse, "true")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(respBytes)
		}))
	server.SetUpstreamHost(mockServer.URL)
	targetUpstreamHost := server.UpstreamHost()
	assert.Equal(t, mockServer.URL, targetUpstreamHost)

	// ==== step 2 GET /config to create root document meta ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)

	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	supportedDocs1 := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729"
	firmwareVersion1 := "CGM4331COM_4.11p7s1_PROD_sey"
	firmwareVersion2 := "CGM4331COM_4.12p7s1_PROD_sey"
	modelName1 := "CGM4331COM"
	partner1 := "comcast"
	schemaVersion1 := "33554433-1.3,33554434-1.3"
	req.Header.Set(common.HeaderSupportedDocs, supportedDocs1)
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion1)
	req.Header.Set(common.HeaderModelName, modelName1)
	req.Header.Set(common.HeaderPartnerID, partner1)
	req.Header.Set(common.HeaderSchemaVersion, schemaVersion1)

	res := ExecuteRequest(req, router).Result()
	assert.NilError(t, err)
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// ==== step 3 add group privatessid ====
	subdocId := "privatessid"

	// post
	url := fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(privatessidBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// get
	req, err = http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.DeepEqual(t, rbytes, privatessidBytes)

	// ==== step 4 add group lan ====
	subdocId = "lan"

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(lanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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

	// ==== step 5 add group wan ====
	subdocId = "wan"

	// post
	url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
	req, err = http.NewRequest("POST", url, bytes.NewReader(wanBytes))
	req.Header.Set("Content-Type", "application/msgpack")
	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
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
	assert.DeepEqual(t, rbytes, wanBytes)

	// ==== step 6 GET /config ====
	req, err = http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	req.Header.Set(common.HeaderSupportedDocs, supportedDocs1)
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion2)
	req.Header.Set(common.HeaderModelName, modelName1)
	req.Header.Set(common.HeaderPartnerID, partner1)
	req.Header.Set(common.HeaderSchemaVersion, schemaVersion1)

	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.Equal(t, res.StatusCode, http.StatusOK)
	assert.NilError(t, err)
	res.Body.Close()
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 3)
	mpart, ok := mparts["privatessid"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, privatessidBytes)
	mpart, ok = mparts["lan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, lanBytes)
	mpart, ok = mparts["wan"]
	assert.Assert(t, ok)
	assert.DeepEqual(t, mpart.Bytes, wanBytes)

	// ==== step 7 verify the states are in_deployment ====
	subdocIds := []string{"privatessid", "lan", "wan"}
	for _, subdocId := range subdocIds {
		url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
		req, err = http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/msgpack")
		assert.NilError(t, err)
		res = ExecuteRequest(req, router).Result()
		rbytes, err = io.ReadAll(res.Body)
		assert.NilError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.DeepEqual(t, rbytes, srcbytesMap[subdocId])
		state, err := strconv.Atoi(res.Header.Get(common.HeaderSubdocumentState))
		assert.NilError(t, err)
		assert.Equal(t, state, common.InDeployment)
	}

	// ==== step 8 update the states ====
	for _, subdocId := range subdocIds {
		notifBody := fmt.Sprintf(`{"namespace": "%v", "application_status": "success", "updated_time": 1635976420, "cpe_doc_version": "984628970", "transaction_uuid": "6ef948f6-cbfa-4620-bde7-8acca1f95ba3_____005CFE970DE53C1"}`, subdocId)
		var m common.EventMessage
		err := json.Unmarshal([]byte(notifBody), &m)
		assert.NilError(t, err)
		fields := make(log.Fields)
		err = db.UpdateDocumentState(server.DatabaseClient, cpeMac, &m, fields)
		assert.NilError(t, err)
	}

	// ==== step 9 verify all states deployed ====
	for _, subdocId := range subdocIds {
		url = fmt.Sprintf("/api/v1/device/%v/document/%v", cpeMac, subdocId)
		req, err = http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/msgpack")
		assert.NilError(t, err)
		res = ExecuteRequest(req, router).Result()
		rbytes, err = io.ReadAll(res.Body)
		assert.NilError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.DeepEqual(t, rbytes, srcbytesMap[subdocId])
		state, err := strconv.Atoi(res.Header.Get(common.HeaderSubdocumentState))
		assert.NilError(t, err)
		assert.Equal(t, state, common.Deployed)
	}
}
