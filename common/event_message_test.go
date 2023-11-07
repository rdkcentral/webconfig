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
package common

import (
	"encoding/json"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestEventMessage(t *testing.T) {
	line1 := `{"device_id":"mac:5c7d7d76cd04","http_status_code":304,"transaction_uuid":"a68c3357-6f60-4a17-b6ac-50f776e75d8f","version":"1607282681"}`
	var m EventMessage
	err := json.Unmarshal([]byte(line1), &m)
	assert.NilError(t, err)
	cpeMac, err := m.Validate(true)
	assert.NilError(t, err)
	expected := "5C7D7D76CD04"
	assert.Equal(t, cpeMac, expected)
	bbytes, err := json.Marshal(m)
	assert.NilError(t, err)
	assert.Assert(t, !strings.Contains(string(bbytes), "application_status"))
	assert.Assert(t, strings.Contains(string(bbytes), "http_status_code"))
	assert.Assert(t, !strings.Contains(string(bbytes), "metrics_agent"))

	m = EventMessage{}
	line2 := `{"application_status":"failed","device_id":"mac:1c9d7233d901","error_code":307,"error_details":"NACK:CcspPandMSsp,Invalid Primary Endpoint IP","namespace":"hotspot","transaction_uuid":"cd6a149e-ac7e-471b-9264-c59354fb62bd","version":"2166722337"}`
	err = json.Unmarshal([]byte(line2), &m)
	assert.NilError(t, err)
	cpeMac, err = m.Validate(true)
	assert.NilError(t, err)
	expected = "1C9D7233D901"
	assert.Equal(t, cpeMac, expected)

	bbytes, err = json.Marshal(m)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(bbytes), "application_status"))
	assert.Assert(t, strings.Contains(string(bbytes), "error_code"))
	assert.Assert(t, strings.Contains(string(bbytes), "error_details"))
	assert.Assert(t, !strings.Contains(string(bbytes), "http_status_code"))
	assert.Assert(t, !strings.Contains(string(bbytes), "metrics_agent"))

	line3 := `{"device_id":"mac:98f781b1089b","reports":[{"url":"https://cpe-config.xdp.comcast.net/api/v1/device/98f781b1089b/config/ble","http_status_code":403,"request_timestamp":1659977051,"version":"NONE","transaction_uuid":"ac93fe18-0be2-43f3-9e2c-9a46dfaee6c1"}]}`
	m = EventMessage{}
	err = json.Unmarshal([]byte(line3), &m)
	assert.NilError(t, err)
	cpeMac, err = m.Validate(true)
	assert.NilError(t, err)
	expected = "98F781B1089B"
	assert.Equal(t, cpeMac, expected)
	bbytes, err = json.Marshal(m)
	assert.Assert(t, !strings.Contains(string(bbytes), "application_status"))
	assert.Assert(t, strings.Contains(string(bbytes), "reports"))

	line4 := `{"application_status":"success","device_id":"mac:8c6a8d5582a4","namespace":"aker","transaction_uuid":"b367ad37-1f8f-4a99-acf1-e6369a9","version":"1123225282"}`
	m = EventMessage{}
	err = json.Unmarshal([]byte(line4), &m)
	assert.NilError(t, err)
	cpeMac, err = m.Validate(true)
	assert.NilError(t, err)
	expected = "8C6A8D5582A4"
	assert.Equal(t, cpeMac, expected)
	bbytes, err = json.Marshal(m)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(bbytes), "application_status"))
	assert.Assert(t, !strings.Contains(string(bbytes), "http_status_code"))
	assert.Assert(t, !strings.Contains(string(bbytes), "error_code"))
	assert.Assert(t, !strings.Contains(string(bbytes), "error_details"))
	assert.Assert(t, !strings.Contains(string(bbytes), "reports"))

	line5 := `{"device_id":"mac:5c7d7d76cd04","http_status_code":304,"transaction_uuid":"a68c3357-6f60-4a17-b6ac-50f776e75d8f","version":"1607282681","metrics_agent":"smoketest"}`
	m = EventMessage{}
	err = json.Unmarshal([]byte(line5), &m)
	assert.NilError(t, err)
	cpeMac, err = m.Validate(true)
	assert.NilError(t, err)
	expected = "5C7D7D76CD04"
	assert.Equal(t, cpeMac, expected)
	bbytes, err = json.Marshal(m)
	assert.NilError(t, err)
	assert.Assert(t, !strings.Contains(string(bbytes), "application_status"))
	assert.Assert(t, strings.Contains(string(bbytes), "http_status_code"))
	assert.Assert(t, strings.Contains(string(bbytes), "metrics_agent"))

	m = EventMessage{}
	line6 := `{"application_status":"failed","device_id":"mac:1c9d7233d901","error_code":307,"error_details":"NACK:CcspPandMSsp,Invalid Primary Endpoint IP","namespace":"hotspot","transaction_uuid":"cd6a149e-ac7e-471b-9264-c59354fb62bd","version":"2166722337","metrics_agent":"smoketest"}`
	err = json.Unmarshal([]byte(line6), &m)
	assert.NilError(t, err)
	cpeMac, err = m.Validate(true)
	assert.NilError(t, err)
	expected = "1C9D7233D901"
	assert.Equal(t, cpeMac, expected)

	bbytes, err = json.Marshal(m)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(bbytes), "application_status"))
	assert.Assert(t, strings.Contains(string(bbytes), "error_code"))
	assert.Assert(t, strings.Contains(string(bbytes), "error_details"))
	assert.Assert(t, !strings.Contains(string(bbytes), "http_status_code"))
	assert.Assert(t, strings.Contains(string(bbytes), "metrics_agent"))
}
