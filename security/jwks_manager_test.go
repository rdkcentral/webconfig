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
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rdkcentral/webconfig/common"
	"gotest.tools/assert"
)

// XPC-43727 real JWKS-backed fixtures: a self-contained RSA key pair and a
// keyfunc.JWKS built from it via keyfunc.NewJSON(), so VerifyApiToken can be
// exercised end-to-end (real signature verification) without any HTTP/JWKS
// server dependency.
const jwksTestKid = "jwks-test-key"

func newJwksTestFixture(t *testing.T) (*rsa.PrivateKey, *keyfunc.JWKS) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)

	b64 := func(b []byte) string {
		return base64.RawURLEncoding.EncodeToString(b)
	}
	eBytes := big.NewInt(int64(privateKey.PublicKey.E)).Bytes()

	jwkJSON := fmt.Sprintf(`{"keys":[{"kty":"RSA","kid":%q,"alg":"RS256","n":%q,"e":%q}]}`,
		jwksTestKid, b64(privateKey.PublicKey.N.Bytes()), b64(eBytes))

	jwks, err := keyfunc.NewJSON(json.RawMessage(jwkJSON))
	assert.NilError(t, err)

	return privateKey, jwks
}

func signJwksTestToken(t *testing.T, privateKey *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = jwksTestKid
	tokenString, err := token.SignedString(privateKey)
	assert.NilError(t, err)
	return tokenString
}

func TestJwksManagerVerifyApiTokenValidCapability(t *testing.T) {
	privateKey, jwks := newJwksTestFixture(t)
	m := &JwksManager{jwks: jwks, apiCapabilities: []string{"webconfig:all"}}

	utcnow := time.Now()
	token := signJwksTestToken(t, privateKey, jwt.MapClaims{
		"capabilities": []interface{}{"webconfig:all"},
		"exp":          utcnow.Add(time.Hour).Unix(),
	})

	ok, err := m.VerifyApiToken(token)
	assert.NilError(t, err)
	assert.Assert(t, ok)
}

func TestJwksManagerVerifyApiTokenMissingCapability(t *testing.T) {
	privateKey, jwks := newJwksTestFixture(t)
	m := &JwksManager{jwks: jwks, apiCapabilities: []string{"webconfig:all"}}

	utcnow := time.Now()
	token := signJwksTestToken(t, privateKey, jwt.MapClaims{
		"capabilities": []interface{}{"some:other:capability"},
		"exp":          utcnow.Add(time.Hour).Unix(),
	})

	ok, err := m.VerifyApiToken(token)
	assert.Assert(t, !ok)
	var noCapsErr common.NoCapabilitiesError
	assert.Assert(t, errors.As(err, &noCapsErr))
}

// XPC-43727 a non-string entry in the capabilities claim must not panic;
// it must be treated as not-capable, same as an absent/mismatched capability.
func TestJwksManagerVerifyApiTokenMalformedCapabilityDoesNotPanic(t *testing.T) {
	privateKey, jwks := newJwksTestFixture(t)
	m := &JwksManager{jwks: jwks, apiCapabilities: []string{"webconfig:all"}}

	utcnow := time.Now()
	token := signJwksTestToken(t, privateKey, jwt.MapClaims{
		"capabilities": []interface{}{12345, "some:other:capability"},
		"exp":          utcnow.Add(time.Hour).Unix(),
	})

	ok, err := m.VerifyApiToken(token)
	assert.Assert(t, !ok)
	var noCapsErr common.NoCapabilitiesError
	assert.Assert(t, errors.As(err, &noCapsErr))
}

func TestJwksManagerVerifyApiTokenInvalidSignature(t *testing.T) {
	privateKey, jwks := newJwksTestFixture(t)
	m := &JwksManager{jwks: jwks, apiCapabilities: []string{"webconfig:all"}}

	utcnow := time.Now()
	token := signJwksTestToken(t, privateKey, jwt.MapClaims{
		"capabilities": []interface{}{"webconfig:all"},
		"exp":          utcnow.Add(time.Hour).Unix(),
	})

	// tamper a character in the middle of the signature segment; avoid the
	// last char, whose low bits may be unused by base64url padding
	parts := []rune(token)
	// find the position of the last '.' to locate the signature segment
	lastDot := -1
	for i, r := range parts {
		if r == '.' {
			lastDot = i
		}
	}
	assert.Assert(t, lastDot >= 0 && lastDot+2 < len(parts))
	mid := lastDot + 1 + (len(parts)-lastDot-1)/2
	if parts[mid] == 'A' {
		parts[mid] = 'B'
	} else {
		parts[mid] = 'A'
	}
	tampered := string(parts)

	ok, err := m.VerifyApiToken(tampered)
	assert.Assert(t, !ok)
	var noCapsErr common.NoCapabilitiesError
	assert.Assert(t, !errors.As(err, &noCapsErr))
}

func TestJwksManagerVerifyApiTokenExpiredToken(t *testing.T) {
	privateKey, jwks := newJwksTestFixture(t)
	m := &JwksManager{jwks: jwks, apiCapabilities: []string{"webconfig:all"}}

	utcnow := time.Now()
	token := signJwksTestToken(t, privateKey, jwt.MapClaims{
		"capabilities": []interface{}{"webconfig:all"},
		"exp":          utcnow.Add(-time.Hour).Unix(),
	})

	ok, err := m.VerifyApiToken(token)
	assert.Assert(t, !ok)
	var noCapsErr common.NoCapabilitiesError
	assert.Assert(t, !errors.As(err, &noCapsErr))
}
