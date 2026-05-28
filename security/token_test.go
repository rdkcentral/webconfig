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
package security

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestLoadingKeyFiles(t *testing.T) {
	sc, err := common.GetTestServerConfig()
	if err != nil {
		panic(err)
	}
	if !sc.GetBoolean("webconfig.jwt.enabled") {
		t.Skip("webconfig.jwt.enabled = false")
	}

	publicKeyFile := "/etc/webconfig/webconfig_key_pub.pem"
	_, err = loadDecodeKey(publicKeyFile)
	assert.NilError(t, err)

	badPublicKeyFile := "/etc/webconfig/webconfig_key_pub.pemx"
	_, err = loadDecodeKey(badPublicKeyFile)
	assert.Assert(t, errors.Is(err, os.ErrNotExist))

	privateKeyFile := "/etc/webconfig/webconfig_key.pem"
	_, err = loadEncodeKey(privateKeyFile)
	assert.NilError(t, err)

	badPrivateKeyFile := "/etc/webconfig/webconfig_key.pemx"
	_, err = loadEncodeKey(badPrivateKeyFile)
	assert.Assert(t, errors.Is(err, os.ErrNotExist))
}

func TestTokenValidation(t *testing.T) {
	if tokenManager == nil {
		t.Skip("webconfig.jwt.enabled = false")
	}

	cpeMac := util.GenerateRandomCpeMac()
	token := tokenManager.Generate(strings.ToLower(cpeMac), 86400)

	// default comcast
	ok, parsedPartner, trust, err := tokenManager.VerifyCpeToken(token, cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, parsedPartner, "comcast")
	assert.Equal(t, trust, 1000)

	// create a partner token
	partner1 := "cox"
	token1 := tokenManager.Generate(strings.ToLower(cpeMac), 86400, partner1)
	ok, parsedPartner, trust, err = tokenManager.VerifyCpeToken(token1, cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, parsedPartner, partner1)
	assert.Equal(t, trust, 1000)

	// create a partner token with non-default trust
	token2 := tokenManager.Generate(strings.ToLower(cpeMac), 86400, partner1, 500)
	ok, parsedPartner, trust, err = tokenManager.VerifyCpeToken(token2, cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, parsedPartner, partner1)
	assert.Equal(t, trust, 500)
}
