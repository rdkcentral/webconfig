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
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestPayloadBuilder(t *testing.T) {
	srcHeader := make(http.Header)
	srcHeader.Add("Destination", "event:subdoc-report/portmapping/mac:044e5a22c9bf/status")
	srcHeader.Add("Content-type", "application/json")

	srcData := util.Dict{
		"device_id":          "mac:044e5a22c9bf",
		"namespace":          "portmapping",
		"application_status": "success",
		"transaction_uuid":   "b09d4cca-ff85-422d-8cc4-7361473d7296_____0059D25BA5CE227",
		"version":            "84257822727189857814737469707926162619",
	}
	sbytes, err := json.Marshal(srcData)
	assert.NilError(t, err)

	_ = common.BuildPayloadAsHttp(http.StatusOK, srcHeader, sbytes)
	_ = common.BuildPayloadAsHttp(http.StatusForbidden, nil, nil)
}
