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
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/rdkcentral/webconfig/common"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-akka/configuration"
	"github.com/google/uuid"
)

const (
	EncodingKeyId = "webconfig_key"
	// sermo lib
	JwtLibIdDefault = 2
)

type ThemisClaims struct {
	KeyId        string   `json:"kid"`
	Mac          string   `json:"mac"`
	PartnerId    string   `json:"partner-id"`
	Serial       string   `json:"serial"`
	Trust        string   `json:"trust"`
	Uuid         string   `json:"uuid"`
	Capabilities []string `json:"capabilities"`
	jwt.StandardClaims
}

type VerifyFunc func(map[string]*rsa.PublicKey, []string, []string, ...string) (bool, error)

type TokenManager struct {
	encodeKey       *rsa.PrivateKey
	decodeKeys      map[string]*rsa.PublicKey
	apiKids         []string
	apiCapabilities []string
	cpeKids         []string
	cpeCapabilities []string
	verifyFn        VerifyFunc
}

func NewTokenManager(conf *configuration.Config) *TokenManager {
	// prepare args for TokenManager
	privateKeyFile := conf.GetString(fmt.Sprintf("webconfig.jwt.kid.%s.private_key_file", EncodingKeyId))

	kids := conf.GetNode("webconfig.jwt.kid").GetObject().GetKeys()
	decodeKeys := map[string]*rsa.PublicKey{}
	for _, kid := range kids {
		keyfile := conf.GetString(fmt.Sprintf("webconfig.jwt.kid.%s.public_key_file", kid))
		dk, err := loadDecodeKey(keyfile)
		if err != nil {
			panic(err)
		}
		decodeKeys[kid] = dk
	}

	fn := VerifyBySermo

	// load the private encoding key
	encodeKey, err := loadEncodeKey(privateKeyFile)
	if err != nil {
		panic(err)
	}

	// default to sermo_jose verifier
	return &TokenManager{
		encodeKey:       encodeKey,
		decodeKeys:      decodeKeys,
		apiKids:         conf.GetStringList("webconfig.jwt.api_token.kids"),
		apiCapabilities: conf.GetStringList("webconfig.jwt.api_token.capabilities"),
		cpeKids:         conf.GetStringList("webconfig.jwt.cpe_token.kids"),
		cpeCapabilities: conf.GetStringList("webconfig.jwt.cpe_token.capabilities"),
		verifyFn:        fn,
	}
}

func loadDecodeKey(keyfile string) (*rsa.PublicKey, error) {
	kbytes, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, common.NewError(err)
	}
	decodeKey, err := jwt.ParseRSAPublicKeyFromPEM(kbytes)
	if err != nil {
		return decodeKey, common.NewError(err)
	}
	return decodeKey, nil
}

func loadEncodeKey(keyfile string) (*rsa.PrivateKey, error) {
	kbytes, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, common.NewError(err)
	}
	encodeKey, err := jwt.ParseRSAPrivateKeyFromPEM(kbytes)
	if err != nil {
		return encodeKey, common.NewError(err)
	}
	return encodeKey, nil
}

func (m *TokenManager) Generate(mac string, ttl int64) string {
	// %% NOTE mac should be lowercase to be consistent with reference doc
	// static themis fields copied from examples in the webconfig confluence
	kid := "webconfig_key"
	serial := "ABCNDGE"
	trust := "1000"
	capUuid := "1234567891234"

	utcnow := time.Now().Unix()

	claims := ThemisClaims{
		kid,
		mac,
		"comcast",
		serial,
		trust,
		capUuid,
		[]string{"x1:issuer:test:.*:all"},
		jwt.StandardClaims{
			Audience:  "XMiDT",
			ExpiresAt: utcnow + ttl,
			Id:        uuid.New().String(),
			IssuedAt:  utcnow,
			Issuer:    "themis",
			NotBefore: utcnow,
			Subject:   "client:supplied",
		},
	}
	method := jwt.GetSigningMethod("RS256")

	token := jwt.NewWithClaims(method, claims)
	// %% note the default Header is { "alg": "RS256", "typ": "JWT" }
	if _, ok := token.Header["typ"]; ok {
		delete(token.Header, "typ")
	}
	token.Header["kid"] = kid

	var tokenString string
	if signedString, err := token.SignedString(m.encodeKey); err == nil {
		tokenString = signedString
	}
	return tokenString
}

func PaddingB64(input string) string {
	output := input
	switch len(input) % 4 {
	case 2:
		output += "=="
	case 3:
		output += "="
	}
	return output
}

func ParseKidFromTokenHeader(tokenString string) (string, error) {
	elements := strings.Split(tokenString, ".")
	if len(elements) != 3 {
		return "", common.NewError(fmt.Errorf("illegal jwt token"))
	}

	var kid string
	rbytes, err := base64.StdEncoding.DecodeString(PaddingB64(elements[0]))
	if err != nil {
		return kid, common.NewError(err)
	}

	headers := map[string]interface{}{}
	if err := json.Unmarshal(rbytes, &headers); err != nil {
		return kid, common.NewError(err)
	}

	rawKid, ok := headers["kid"]
	if !ok {
		return kid, common.NewError(common.NotOK)
	}
	kid, ok = rawKid.(string)
	if !ok {
		return kid, common.NewError(common.NotOK)
	}

	return kid, nil
}

func (m *TokenManager) VerifyApiToken(token string) (bool, error) {
	return m.verifyFn(m.decodeKeys, m.apiKids, m.apiCapabilities, token)
}

func (m *TokenManager) VerifyCpeToken(token string, mac string) (bool, error) {
	return m.verifyFn(m.decodeKeys, m.cpeKids, m.cpeCapabilities, token, mac)
}

func (m *TokenManager) SetVerifyFunc(fn VerifyFunc) {
	m.verifyFn = fn
}
