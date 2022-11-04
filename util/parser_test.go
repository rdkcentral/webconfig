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
package util

import (
	"encoding/json"
	"net/http"
	"testing"

	"gotest.tools/assert"
)

func TestParseWebconfigResponseMessage(t *testing.T) {
	srcHeader := make(http.Header)
	srcHeader.Add("Destination", "event:subdoc-report/portmapping/mac:044e5a22c9bf/status")
	srcHeader.Add("Content-type", "application/json")
	srcHeader.Add("Content-length", "120")

	srcData := Dict{
		"device_id":          "mac:044e5a22c9bf",
		"namespace":          "portmapping",
		"application_status": "success",
		"transaction_uuid":   "b09d4cca-ff85-422d-8cc4-7361473d7296_____0059D25BA5CE227",
		"version":            "84257822727189857814737469707926162619",
	}
	sbytes, err := json.Marshal(srcData)
	assert.NilError(t, err)

	srcMessage := BuildHttp(srcHeader, sbytes)
	tgtHeader, tbytes := ParseHttp(srcMessage)
	var tgtData Dict
	err = json.Unmarshal(tbytes, &tgtData)
	assert.NilError(t, err)
	assert.DeepEqual(t, srcHeader, tgtHeader)
	assert.DeepEqual(t, srcData, tgtData)
}

func TestParseWebconfigResponseMessageMultipleHeaders(t *testing.T) {
	srcHeader := make(http.Header)
	srcHeader.Add("X-Color", "color:red")
	srcHeader.Add("Destination", "event:subdoc-report/portmapping/mac:044e5a22c9bf/status")
	srcHeader.Add("X-Color", "color:orange")
	srcHeader.Add("Content-type", "application/json")
	srcHeader.Add("X-Color", "color:yellow:green")
	srcHeader.Add("Content-length", "120")
	srcHeader.Add("X-Color", "color:blue:indigo:violet")

	srcData := Dict{
		"device_id":          "mac:044e5a22c9bf",
		"namespace":          "portmapping",
		"application_status": "success",
		"transaction_uuid":   "b09d4cca-ff85-422d-8cc4-7361473d7296_____0059D25BA5CE227",
		"version":            "84257822727189857814737469707926162619",
	}

	sbytes, err := json.Marshal(srcData)
	assert.NilError(t, err)

	srcMessage := BuildHttp(srcHeader, sbytes)
	tgtHeader, tbytes := ParseHttp(srcMessage)
	var tgtData Dict
	err = json.Unmarshal(tbytes, &tgtData)
	assert.NilError(t, err)
	assert.DeepEqual(t, srcHeader, tgtHeader)
	assert.DeepEqual(t, srcData, tgtData)
}

func TestParseWebconfigGetMessage(t *testing.T) {
	srcHeader := make(http.Header)
	srcHeader.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6IndlYmNvbmZpZ19rZXkifQ")
	srcHeader.Add("If-None-Match", "1670697802,1426217395,1874871438,1102611645,2845667022,773805505,2605450606,2192982180")
	srcHeader.Add("Schema-Version", "v1.0")
	srcHeader.Add("Transaction-Id", "a8309ac0-dc4b-46d7-9c52-0c55e7aa07fb_____005D9CC4F03BA57")
	srcHeader.Add("X-System-Boot-Time", "1666332132")
	srcHeader.Add("X-System-Current-Time", "1666335732")
	srcHeader.Add("X-System-Firmware-Version", "TG3482PC2_5.3p11s1_PROD_sey")
	srcHeader.Add("X-System-Model-Name", "TG3482G")
	srcHeader.Add("X-System-Product-Class", "XB6")
	srcHeader.Add("X-System-Ready-Time", "1666332732")
	srcHeader.Add("X-System-Status", "Operational")
	srcHeader.Add("X-System-Supported-Docs", "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729")

	srcMessage := BuildHttp(srcHeader, nil)
	tgtHeader, tbytes := ParseHttp(srcMessage)
	assert.Equal(t, len(tbytes), 0)
	assert.DeepEqual(t, srcHeader, tgtHeader)
}
