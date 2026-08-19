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
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestApiTokenAuthSecureDefaults(t *testing.T) {
	// Admin write endpoints (document, rootdocument, poke, reference) MUST
	// be guarded by ApiMiddleware out of the box. Regression guard for f003.
	assert.Assert(t, serverApiTokenAuthEnabledDefault, "server API token auth must default to enabled")
	assert.Assert(t, !configApiTokenAuthEnabledDefault, "config API token auth must default to disabled")
	assert.Assert(t, deviceApiTokenAuthEnabledDefault, "device API token auth must default to enabled")
}

func TestConfigEndpointRemainsUnauthenticatedByDefault(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	assert.Assert(t, !server.ConfigApiTokenAuthEnabled())
	router := server.GetRouter(false)

	req, err := http.NewRequest("GET", "/config", nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
}

func TestConfigEndpointRequiresApiTokenWhenEnabled(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.SetConfigApiTokenAuthEnabled(true)
	assert.Assert(t, server.ConfigApiTokenAuthEnabled())
	router := server.GetRouter(false)

	req, err := http.NewRequest("GET", "/config", nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)
}

func TestApiMiddlewareSuppressesConfigRequestLogs(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.SetConfigApiTokenAuthEnabled(true)
	router := server.GetRouter(false)

	var logs strings.Builder
	previousOutput := log.StandardLogger().Out
	previousLevel := log.StandardLogger().Level
	log.SetOutput(&logs)
	log.SetLevel(log.InfoLevel)
	defer func() {
		log.SetOutput(previousOutput)
		log.SetLevel(previousLevel)
	}()

	req, err := http.NewRequest("GET", "/config", nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)
	assert.Assert(t, !strings.Contains(logs.String(), "Request started"))
	assert.Assert(t, !strings.Contains(logs.String(), "Request finished"))

	logs.Reset()
	req, err = http.NewRequest("GET", "/other", nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, server.ApiMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)
	assert.Assert(t, strings.Contains(logs.String(), "Request started"))
	assert.Assert(t, strings.Contains(logs.String(), "Request finished"))
}

func TestWebconfigServerSetterGetter(t *testing.T) {
	server := NewWebconfigServer(sc, true)

	// factory reset flag
	enabled := true
	server.SetFactoryResetEnabled(enabled)
	assert.Equal(t, server.FactoryResetEnabled(), enabled)
	enabled = false
	server.SetFactoryResetEnabled(enabled)
	assert.Equal(t, server.FactoryResetEnabled(), enabled)

	// server api token auth
	enabled = true
	server.SetServerApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.ServerApiTokenAuthEnabled(), enabled)
	enabled = false
	server.SetServerApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.ServerApiTokenAuthEnabled(), enabled)

	// config api token auth
	enabled = true
	server.SetConfigApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.ConfigApiTokenAuthEnabled(), enabled)
	enabled = false
	server.SetConfigApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.ConfigApiTokenAuthEnabled(), enabled)

	// device api token auth
	enabled = true
	server.SetDeviceApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.DeviceApiTokenAuthEnabled(), enabled)
	enabled = false
	server.SetDeviceApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.DeviceApiTokenAuthEnabled(), enabled)

	// token api
	enabled = true
	server.SetTokenApiEnabled(enabled)
	assert.Equal(t, server.TokenApiEnabled(), enabled)
	enabled = false
	server.SetTokenApiEnabled(enabled)
	assert.Equal(t, server.TokenApiEnabled(), enabled)

	// kafka
	enabled = true
	server.SetKafkaEnabled(enabled)
	assert.Equal(t, server.KafkaEnabled(), enabled)
	enabled = false
	server.SetKafkaEnabled(enabled)
	assert.Equal(t, server.KafkaEnabled(), enabled)

	// upstream
	enabled = true
	server.SetUpstreamEnabled(enabled)
	assert.Equal(t, server.UpstreamEnabled(), enabled)
	enabled = false
	server.SetUpstreamEnabled(enabled)
	assert.Equal(t, server.UpstreamEnabled(), enabled)

	// app name
	name := "foo"
	server.SetAppName(name)
	assert.Equal(t, server.AppName(), name)
	name = "bar"
	server.SetAppName(name)
	assert.Equal(t, server.AppName(), name)

	// validate mac
	enabled = true
	server.SetValidateMacEnabled(enabled)
	assert.Equal(t, server.ValidateMacEnabled(), enabled)
	enabled = false
	server.SetValidateMacEnabled(enabled)
	assert.Equal(t, server.ValidateMacEnabled(), enabled)

	// validate valid partners
	validPartners := []string{"vendor1", "partner2", "company3"}
	server.SetValidPartners(validPartners)
	assert.DeepEqual(t, server.ValidPartners(), validPartners)
	validPartners = []string{"name3", "name4", "name5"}
	server.SetValidPartners(validPartners)
	assert.DeepEqual(t, server.ValidPartners(), validPartners)

	// get profiles from upstream
	enabled = true
	server.SetUpstreamProfilesEnabled(enabled)
	assert.Equal(t, server.UpstreamProfilesEnabled(), enabled)
	enabled = false
	server.SetUpstreamProfilesEnabled(enabled)
	assert.Equal(t, server.UpstreamProfilesEnabled(), enabled)

	// enforce strict query parameters validation
	enabled = true
	server.SetQueryParamsValidationEnabled(enabled)
	assert.Equal(t, server.QueryParamsValidationEnabled(), enabled)
	enabled = false
	server.SetQueryParamsValidationEnabled(enabled)
	assert.Equal(t, server.QueryParamsValidationEnabled(), enabled)

	//configure trust level
	trust := 1000
	server.SetMinTrust(trust)
	assert.Equal(t, server.MinTrust(), trust)
	trust = 500
	server.SetMinTrust(trust)
	assert.Equal(t, server.MinTrust(), trust)

	x := true
	server.SetFilterOutputByBitmapEnabled(x)
	assert.Assert(t, server.FilterOutputByBitmapEnabled())
	x = false
	server.SetFilterOutputByBitmapEnabled(x)
	assert.Assert(t, !server.FilterOutputByBitmapEnabled())

	validSubdocIdMap := map[string]int{
		"red":    1,
		"orange": 2,
		"yellow": 3,
		"green":  4,
	}
	server.SetValidSubdocIdMap(validSubdocIdMap)
	assert.DeepEqual(t, validSubdocIdMap, server.ValidSubdocIdMap())

}
