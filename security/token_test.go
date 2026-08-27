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
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// XPC-42727 a token with an invalid signature must fail as an authentication
// error, never misclassified as ErrNoCapabilities, even when its (untrusted)
// claims lack the required capability
func TestVerifyTokenRejectsInvalidSignatureBeforeCapabilities(t *testing.T) {
	if tokenManager == nil {
		t.Skip("webconfig.jwt.enabled = false")
	}

	cpeMac := util.GenerateRandomCpeMac()
	token := tokenManager.Generate(strings.ToLower(cpeMac), 86400)

	parts := strings.Split(token, ".")
	assert.Equal(t, len(parts), 3)
	// flip a character in the middle of the signature to invalidate it;
	// avoid the last char, whose low bits may be unused by base64url padding
	sig := []rune(parts[2])
	mid := len(sig) / 2
	if sig[mid] == 'A' {
		sig[mid] = 'B'
	} else {
		sig[mid] = 'A'
	}
	tampered := parts[0] + "." + parts[1] + "." + string(sig)

	requiredCapabilities := []string{"capability-not-present-in-token"}
	ok, _, _, verr := VerifyToken(tokenManager.decodeKeys, tokenManager.cpeKids, requiredCapabilities, tampered, cpeMac)
	assert.Assert(t, !ok)
	assert.Assert(t, !errors.Is(verr, common.ErrNoCapabilities))
}

// XPC-43727 a validly signed token whose partner-id claim is not a string
// must fail as a controlled authentication error, never panic
func TestVerifyTokenRejectsMalformedPartnerIdClaim(t *testing.T) {
	if tokenManager == nil {
		t.Skip("webconfig.jwt.enabled = false")
	}

	cpeMac := util.GenerateRandomCpeMac()
	utcnow := time.Now()
	claims := jwt.MapClaims{
		"mac":        strings.ToLower(cpeMac),
		"partner-id": 12345,
		"trust":      1000,
		"exp":        utcnow.Add(time.Hour).Unix(),
	}
	method := jwt.GetSigningMethod("RS256")
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = EncodingKeyId
	tokenString, err := token.SignedString(tokenManager.encodeKey)
	assert.NilError(t, err)

	ok, _, _, verr := VerifyToken(tokenManager.decodeKeys, tokenManager.cpeKids, nil, tokenString, cpeMac)
	assert.Assert(t, !ok)
	assert.ErrorContains(t, verr, "partner-id")
}

// XPC-43727 if an operator configures webconfig.jwt.cpe_token.capabilities,
// VerifyToken enforces it correctly for a real signed CPE token: a matching
// capability succeeds, a non-matching capability fails as ErrNoCapabilities.
// The shipped sample config leaves cpe_token.capabilities empty by design
// (CPE authorization is governed by trust, not capabilities); this proves the
// capability-check code path itself works without changing that
// configuration.
func TestVerifyTokenEnforcesCpeCapabilitiesWhenConfigured(t *testing.T) {
	if tokenManager == nil {
		t.Skip("webconfig.jwt.enabled = false")
	}

	cpeMac := util.GenerateRandomCpeMac()
	sign := func(capabilities []string) string {
		claims := jwt.MapClaims{
			"mac":          strings.ToLower(cpeMac),
			"capabilities": capabilities,
			"exp":          time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)
		token.Header["kid"] = EncodingKeyId
		tokenString, err := token.SignedString(tokenManager.encodeKey)
		assert.NilError(t, err)
		return tokenString
	}

	requiredCapabilities := []string{"cpe:config:read"}

	matching := sign([]string{"cpe:config:read"})
	ok, _, _, err := VerifyToken(tokenManager.decodeKeys, tokenManager.cpeKids, requiredCapabilities, matching, cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, ok)

	nonMatching := sign([]string{"cpe:other"})
	ok, _, _, err = VerifyToken(tokenManager.decodeKeys, tokenManager.cpeKids, requiredCapabilities, nonMatching, cpeMac)
	assert.Assert(t, !ok)
	assert.Assert(t, errors.Is(err, common.ErrNoCapabilities))
}

// XPC-42727 an expired token must fail as an authentication error, never
// misclassified as ErrNoCapabilities, even when its (untrusted) claims lack
// the required capability
func TestVerifyTokenRejectsExpiredTokenBeforeCapabilities(t *testing.T) {
	if tokenManager == nil {
		t.Skip("webconfig.jwt.enabled = false")
	}

	cpeMac := util.GenerateRandomCpeMac()
	utcnow := time.Now()
	claims := ThemisClaims{
		KeyId:     EncodingKeyId,
		Mac:       strings.ToLower(cpeMac),
		PartnerId: "comcast",
		Trust:     1000,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(utcnow.Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(utcnow.Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(utcnow.Add(-2 * time.Hour)),
		},
	}
	method := jwt.GetSigningMethod("RS256")
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = EncodingKeyId
	tokenString, err := token.SignedString(tokenManager.encodeKey)
	assert.NilError(t, err)

	requiredCapabilities := []string{"capability-not-present-in-token"}
	ok, _, _, verr := VerifyToken(tokenManager.decodeKeys, tokenManager.cpeKids, requiredCapabilities, tokenString, cpeMac)
	assert.Assert(t, !ok)
	assert.Assert(t, !errors.Is(verr, common.ErrNoCapabilities))
}
