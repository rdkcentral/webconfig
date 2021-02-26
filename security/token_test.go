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
	"testing"

	"gotest.tools/assert"
)

func TestLoadingKeyFiles(t *testing.T) {
	privateKeyFileName := "/tmp/private.pem"
	publicKeyFileName := "/tmp/public.pem"

	privateKey, err := GenerateKeyPairAsFiles(privateKeyFileName, publicKeyFileName)
	assert.NilError(t, err)

	readPublicKey, err := loadDecodeKey(publicKeyFileName)
	assert.NilError(t, err)
	assert.Assert(t, privateKey.PublicKey.Equal(readPublicKey))

	badPublicKeyFileName := "/tmp/private.pemx"
	_, err = loadDecodeKey(badPublicKeyFileName)
	assert.Assert(t, errors.Is(err, os.ErrNotExist))

	readPrivateKey, err := loadEncodeKey(privateKeyFileName)
	assert.NilError(t, err)
	assert.Assert(t, privateKey.Equal(readPrivateKey))

	badPrivateKeyFileName := "/tmp/public.pemx"
	_, err = loadEncodeKey(badPrivateKeyFileName)
	assert.Assert(t, errors.Is(err, os.ErrNotExist))
}
