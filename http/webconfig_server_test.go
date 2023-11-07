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
	"testing"

	"gotest.tools/assert"
)

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
}
