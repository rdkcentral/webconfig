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
	"encoding/base64"
	"os"
	"testing"

	"gotest.tools/assert"
)

func TestDecryption(t *testing.T) {
	dec, err := testCodec.Decrypt(Encrypted1a)
	assert.NilError(t, err)
	assert.Equal(t, dec, Plaintext1)
	dec, err = testCodec.Decrypt(Encrypted1b)
	assert.NilError(t, err)
	assert.Equal(t, dec, Plaintext1)

	dec, err = testCodec.Decrypt(Encrypted2a)
	assert.NilError(t, err)
	assert.Equal(t, dec, Plaintext2)
	dec, err = testCodec.Decrypt(Encrypted2b)
	assert.NilError(t, err)
	assert.Equal(t, dec, Plaintext2)

	dec, err = testCodec.Decrypt(Encrypted3a)
	assert.NilError(t, err)
	assert.Equal(t, dec, Plaintext3)
	dec, err = testCodec.Decrypt(Encrypted3b)
	assert.NilError(t, err)
	assert.Equal(t, dec, Plaintext3)
}

func TestXpcKeyFuncs(t *testing.T) {
	xpckey := GetRandomXpcKey()
	assert.Assert(t, len(xpckey) > 0)

	// verify by checking if it is base64-decodable
	bbytes, err := base64.StdEncoding.DecodeString(xpckey)
	assert.NilError(t, err)

	// because we hard coded to use 16 bytes
	assert.Equal(t, len(bbytes), 16)
}

func TestNoKeyCodec(t *testing.T) {
	err := os.Unsetenv("XPC_KEY")
	assert.NilError(t, err)

	nokeyCodec, err := NewAesCodec()
	assert.Assert(t, err != nil)

	srcText := "helloworld"
	encrypted, err := nokeyCodec.Encrypt(srcText)
	assert.NilError(t, err)
	decrypted, err := nokeyCodec.Decrypt(encrypted)
	assert.NilError(t, err)
	assert.Equal(t, srcText, decrypted)
}
