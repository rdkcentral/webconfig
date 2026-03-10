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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestSupplementaryPrecookConfigFlags(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)

	// Test default values
	assert.Equal(t, server.SupplementaryPrecookEnabled(), false)
	assert.Equal(t, server.SupplementaryPrecookStateTTLDays(), 7)

	// Test setter for SupplementaryPrecookEnabled
	server.SetSupplementaryPrecookEnabled(true)
	assert.Equal(t, server.SupplementaryPrecookEnabled(), true)

	server.SetSupplementaryPrecookEnabled(false)
	assert.Equal(t, server.SupplementaryPrecookEnabled(), false)

	// Test setter for SupplementaryPrecookStateTTLDays
	server.SetSupplementaryPrecookStateTTLDays(14)
	assert.Equal(t, server.SupplementaryPrecookStateTTLDays(), 14)

	server.SetSupplementaryPrecookStateTTLDays(0)
	assert.Equal(t, server.SupplementaryPrecookStateTTLDays(), 0)
}

func TestSupplementaryPrecookStateDeployed(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== setup telemetry subdoc with state=Deployed ====
	telemetryBytes := common.RandomBytes(100, 150)
	telemetryVersion := util.GetMurmur3Hash(telemetryBytes)
	telemetryState := common.Deployed
	telemetryUpdatedTime := int(time.Now().UnixMilli())

	telemetrySubdoc := common.NewSubDocument(telemetryBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// ==== verify that MultipartSupplementaryHandler returns 304 ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)
}

func TestSupplementaryPrecookStatePendingDownload(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== create telemetry payload from mock profile response ====
	mockProfileBytes := []byte(mockProfileResponse)
	telemetryVersion := util.GetMurmur3Hash(mockProfileBytes)
	telemetryState := common.PendingDownload
	telemetryUpdatedTime := int(time.Now().UnixMilli())

	telemetrySubdoc := common.NewSubDocument(mockProfileBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// ==== verify that MultipartSupplementaryHandler returns cached data ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// Verify response is multipart
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	_, ok := mparts["telemetry"]
	assert.Assert(t, ok)
}

func TestSupplementaryPrecookStateInDeployment(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== create telemetry payload from mock profile response ====
	mockProfileBytes := []byte(mockProfileResponse)
	telemetryVersion := util.GetMurmur3Hash(mockProfileBytes)
	telemetryState := common.InDeployment
	telemetryUpdatedTime := int(time.Now().UnixMilli())

	telemetrySubdoc := common.NewSubDocument(mockProfileBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// ==== verify that MultipartSupplementaryHandler returns cached data ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// Verify response is multipart
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	_, ok := mparts["telemetry"]
	assert.Assert(t, ok)
}

func TestSupplementaryPrecookStateFailure(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== create telemetry payload from mock profile response ====
	mockProfileBytes := []byte(mockProfileResponse)
	telemetryVersion := util.GetMurmur3Hash(mockProfileBytes)
	telemetryState := common.Failure
	telemetryUpdatedTime := int(time.Now().UnixMilli())
	errorCode := 500
	errorDetails := "test error"

	telemetrySubdoc := common.NewSubDocument(mockProfileBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, &errorCode, &errorDetails)
	fields := make(log.Fields)
	err := server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// ==== verify that MultipartSupplementaryHandler returns cached data ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// Verify response is multipart
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	_, ok := mparts["telemetry"]
	assert.Assert(t, ok)
}

func TestSupplementaryPrecookDisabled(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Ensure supplementary precook feature is disabled
	server.SetSupplementaryPrecookEnabled(false)

	// ==== setup mock server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockProfileResponse))
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== setup telemetry subdoc with state=Deployed ====
	telemetryBytes := common.RandomBytes(100, 150)
	telemetryVersion := util.GetMurmur3Hash(telemetryBytes)
	telemetryState := common.Deployed
	telemetryUpdatedTime := int(time.Now().UnixMilli())

	telemetrySubdoc := common.NewSubDocument(telemetryBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// ==== verify that MultipartSupplementaryHandler uses normal flow (not 304) ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()

	// Should not return 304, should fetch from mock server
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// Verify response is multipart
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
}

func TestSupplementaryPrecookStateUpdate(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== create telemetry payload with state=PendingDownload ====
	mockProfileBytes := []byte(mockProfileResponse)
	telemetryVersion := util.GetMurmur3Hash(mockProfileBytes)
	telemetryState := common.PendingDownload
	telemetryUpdatedTime := int(time.Now().UnixMilli())

	telemetrySubdoc := common.NewSubDocument(mockProfileBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// ==== verify initial state is PendingDownload ====
	subdoc, err := server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.PendingDownload)

	// ==== make request to MultipartSupplementaryHandler ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// Verify response is multipart
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)

	// ==== verify state is updated to InDeployment ====
	subdoc, err = server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.InDeployment)
}

func TestSupplementaryPrecookStateUpdateFromFailure(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== create telemetry payload with state=Failure ====
	mockProfileBytes := []byte(mockProfileResponse)
	telemetryVersion := util.GetMurmur3Hash(mockProfileBytes)
	telemetryState := common.Failure
	telemetryUpdatedTime := int(time.Now().UnixMilli())
	errorCode := 500
	errorDetails := "test error"

	telemetrySubdoc := common.NewSubDocument(mockProfileBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, &errorCode, &errorDetails)
	fields := make(log.Fields)
	err := server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// ==== verify initial state is Failure ====
	subdoc, err := server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Failure)

	// ==== make request to MultipartSupplementaryHandler ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// Verify response is multipart
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)

	// ==== verify state is updated to InDeployment ====
	subdoc, err = server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.InDeployment)
}

func TestSupplementaryPrecookStateNotUpdatedWhenDeployed(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== create telemetry payload with state=Deployed ====
	telemetryBytes := common.RandomBytes(100, 150)
	telemetryVersion := util.GetMurmur3Hash(telemetryBytes)
	telemetryState := common.Deployed
	telemetryUpdatedTime := int(time.Now().UnixMilli())

	telemetrySubdoc := common.NewSubDocument(telemetryBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// ==== verify initial state is Deployed ====
	subdoc, err := server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Deployed)

	// ==== make request to MultipartSupplementaryHandler ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusNotModified)

	// ==== verify state is still Deployed (not updated) ====
	subdoc, err = server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Deployed)
}

func TestSupplementaryPrecookStateDeployedWithExpiredTTL(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup mock xconf server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockProfileResponse))
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== setup telemetry subdoc with state=Deployed but expired TTL ====
	mockProfileBytes := []byte(mockProfileResponse)
	telemetryVersion := util.GetMurmur3Hash(mockProfileBytes)
	telemetryState := common.Deployed
	telemetryUpdatedTime := int(time.Now().UnixMilli())

	// Set expiry to a past time (1 hour ago in milliseconds)
	expiredTime := int(time.Now().Add(-1 * time.Hour).UnixMilli())

	telemetrySubdoc := common.NewSubDocument(mockProfileBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, nil, nil)
	telemetrySubdoc.SetExpiry(&expiredTime)

	fields := make(log.Fields)
	err := server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// ==== verify the subdoc has state=Deployed and expired expiry ====
	subdoc, err := server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Deployed)
	assert.Assert(t, subdoc.Expiry() != nil)
	assert.Assert(t, *subdoc.Expiry() < int(time.Now().UnixMilli()))

	// ==== make request to MultipartSupplementaryHandler ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()

	// ==== expectation: should NOT return 304, should fetch from xconf ====
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// Verify response is multipart from xconf
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	_, ok := mparts["telemetry"]
	assert.Assert(t, ok)
}

func TestSupplementaryPrecookExpiryDeletedOnStateTransition(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== create root document for labels ====
	queryParams := "model=TG1682G&partner=comcast"
	fwVersion := "TG1682_3.14p9s6_PROD_sey"
	modelName := "TG1682G"
	partnerId := "comcast"
	rootDoc := common.NewRootDocument(0, fwVersion, modelName, partnerId, "", "", queryParams, "", "")
	err := server.SetRootDocument(cpeMac, rootDoc)
	assert.NilError(t, err)

	// ==== create telemetry payload from mock profile response with state=PendingDownload and expiry set ====
	mockProfileBytes := []byte(mockProfileResponse)
	telemetryVersion := util.GetMurmur3Hash(mockProfileBytes)
	telemetryState := common.PendingDownload
	telemetryUpdatedTime := int(time.Now().UnixMilli())
	futureExpiry := int(time.Now().Add(24 * time.Hour).UnixMilli())

	telemetrySubdoc := common.NewSubDocument(mockProfileBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, nil, nil)
	telemetrySubdoc.SetExpiry(&futureExpiry)
	fields := make(log.Fields)
	err = server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// Verify expiry is set before the request
	subdocBefore, err := server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Assert(t, subdocBefore.Expiry() != nil)
	assert.Equal(t, *subdocBefore.Expiry(), futureExpiry)

	// ==== make request to MultipartSupplementaryHandler ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// Verify response is multipart
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	_, ok := mparts["telemetry"]
	assert.Assert(t, ok)

	// ==== verify state was updated to InDeployment ====
	subdocAfter, err := server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Equal(t, *subdocAfter.State(), common.InDeployment)

	// ==== verify expiry was deleted ====
	assert.Assert(t, subdocAfter.Expiry() == nil)
}

func TestSupplementaryPrecookErrorFieldsDeletedOnStateTransition(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// Enable supplementary precook feature
	server.SetSupplementaryPrecookEnabled(true)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== create root document for labels ====
	queryParams := "model=TG1682G&partner=comcast"
	fwVersion := "TG1682_3.14p9s6_PROD_sey"
	modelName := "TG1682G"
	partnerId := "comcast"
	rootDoc := common.NewRootDocument(0, fwVersion, modelName, partnerId, "", "", queryParams, "", "")
	err := server.SetRootDocument(cpeMac, rootDoc)
	assert.NilError(t, err)

	// ==== create telemetry payload with state=Failure (state=4) and error fields set ====
	mockProfileBytes := []byte(mockProfileResponse)
	telemetryVersion := util.GetMurmur3Hash(mockProfileBytes)
	telemetryState := common.Failure
	telemetryUpdatedTime := int(time.Now().UnixMilli())
	errorCode := 204
	errorDetails := "failed_retrying:Error unsupported namespace"
	futureExpiry := int(time.Now().Add(24 * time.Hour).UnixMilli())

	telemetrySubdoc := common.NewSubDocument(mockProfileBytes, &telemetryVersion, &telemetryState, &telemetryUpdatedTime, &errorCode, &errorDetails)
	telemetrySubdoc.SetExpiry(&futureExpiry)
	fields := make(log.Fields)
	err = server.SetSubDocument(cpeMac, "telemetry", telemetrySubdoc, fields)
	assert.NilError(t, err)

	// Verify expiry and error fields are set before the request
	subdocBefore, err := server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Assert(t, subdocBefore.Expiry() != nil)
	assert.Equal(t, *subdocBefore.Expiry(), futureExpiry)
	assert.Assert(t, subdocBefore.ErrorCode() != nil)
	assert.Equal(t, *subdocBefore.ErrorCode(), errorCode)
	assert.Assert(t, subdocBefore.ErrorDetails() != nil)
	assert.Equal(t, *subdocBefore.ErrorDetails(), errorDetails)

	// ==== make request to MultipartSupplementaryHandler ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// Verify response is multipart
	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	_, ok := mparts["telemetry"]
	assert.Assert(t, ok)

	// ==== verify state was updated to InDeployment ====
	subdocAfter, err := server.GetSubDocument(cpeMac, "telemetry")
	assert.NilError(t, err)
	assert.Equal(t, *subdocAfter.State(), common.InDeployment)

	// ==== verify expiry and error fields were deleted ====
	assert.Assert(t, subdocAfter.Expiry() == nil)
	// Note: Cassandra returns default values (0, "") for error_code and error_details after DELETE
	assert.Assert(t, subdocAfter.ErrorCode() != nil)
	assert.Equal(t, *subdocAfter.ErrorCode(), 0)
	assert.Assert(t, subdocAfter.ErrorDetails() != nil)
	assert.Equal(t, *subdocAfter.ErrorDetails(), "")
}
