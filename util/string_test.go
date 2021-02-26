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
	"testing"

	"gotest.tools/assert"
)

func TestString(t *testing.T) {
	s := "112233445566"
	c := ToColonMac(s)
	expected := "11:22:33:44:55:66"
	assert.Equal(t, c, expected)
}

func TestValidateMac(t *testing.T) {
	mac := "001122334455"
	assert.Assert(t, ValidateMac(mac))

	mac = "4444ABCDEF01"
	assert.Assert(t, ValidateMac(mac))

	mac = "00112233445Z"
	assert.Assert(t, !ValidateMac(mac))

	mac = "001122334455Z"
	assert.Assert(t, !ValidateMac(mac))

	mac = "0H1122334455"
	assert.Assert(t, !ValidateMac(mac))

	for i := 0; i < 10; i++ {
		mac := GenerateRandomCpeMac()
		assert.Assert(t, ValidateMac(mac))
	}
}

func TestGetAuditId(t *testing.T) {
	auditId := GetAuditId()
	assert.Equal(t, len(auditId), 32)
}
