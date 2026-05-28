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
	"os"
	"strings"
	"time"

	"github.com/go-akka/configuration"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
)

const (
	EncodingKeyId = "webconfig_key"
)

type ThemisClaims struct {
	KeyId        string   `json:"kid"`
	Mac          string   `json:"mac"`
	PartnerId    string   `json:"partner-id"`
	Serial       string   `json:"serial"`
	Trust        int      `json:"trust"`
	Uuid         string   `json:"uuid"`
	Capabilities []string `json:"capabilities"`
	jwt.RegisteredClaims
}

type VerifyFunc func(map[string]*rsa.PublicKey, []string, []string, ...string) (bool, string, int, error)

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
	panicExitEnabled := conf.GetBoolean("webconfig.panic_exit_enabled", false)

	// prepare args for TokenManager
	privateKeyFile := conf.GetString(fmt.Sprintf("webconfig.jwt.kid.%s.private_key_file", EncodingKeyId))

	kids := conf.GetNode("webconfig.jwt.kid").GetObject().GetKeys()
	decodeKeys := map[string]*rsa.PublicKey{}
	for _, kid := range kids {
		keyfile := conf.GetString(fmt.Sprintf("webconfig.jwt.kid.%s.public_key_file", kid))
		dk, err := loadDecodeKey(keyfile)
		if err != nil {
			if panicExitEnabled {
				panic(err)
			} else {
				fmt.Printf("WARNING %v\n", err)
			}
		}
		decodeKeys[kid] = dk
	}

	fn := VerifyToken

	// load the private encoding key
	encodeKey, err := loadEncodeKey(privateKeyFile)
	if err != nil {
		if panicExitEnabled {
			panic(err)
		} else {
			fmt.Printf("WARNING %v\n", err)
		}
	}

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
	kbytes, err := os.ReadFile(keyfile)
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
	kbytes, err := os.ReadFile(keyfile)
	if err != nil {
		return nil, common.NewError(err)
	}
	encodeKey, err := jwt.ParseRSAPrivateKeyFromPEM(kbytes)
	if err != nil {
		return encodeKey, common.NewError(err)
	}
	return encodeKey, nil
}

// TODO this is not an officially supported function.
func (m *TokenManager) Generate(mac string, ttl int64, itfs ...interface{}) string {
	// %% NOTE mac should be lowercase to be consistent with reference doc
	// static themis fields copied from examples in the webconfig confluence
	kid := "webconfig_key"
	serial := "ABCNDGE"
	trust := 1000
	capUuid := "1234567891234"
	capabilities := []string{"x1:issuer:test:.*:all"}
	partner := "comcast"

	for _, itf := range itfs {
		switch ty := itf.(type) {
		case string:
			partner = ty
		case int:
			trust = ty
		}
	}

	utcnow := time.Now()

	claims := ThemisClaims{
		KeyId:        kid,
		Mac:          mac,
		PartnerId:    partner,
		Serial:       serial,
		Trust:        trust,
		Uuid:         capUuid,
		Capabilities: capabilities,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{"XMiDT"},
			ExpiresAt: jwt.NewNumericDate(utcnow.Add(24 * time.Hour)),
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(utcnow),
			Issuer:    "themis",
			NotBefore: jwt.NewNumericDate(utcnow),
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
		return kid, common.NewError(common.ErrNotOK)
	}
	kid, ok = rawKid.(string)
	if !ok {
		return kid, common.NewError(common.ErrNotOK)
	}

	return kid, nil
}

func (m *TokenManager) VerifyApiToken(token string) (bool, error) {
	ok, _, _, err := m.verifyFn(m.decodeKeys, m.apiKids, m.apiCapabilities, token)
	if err != nil {
		return ok, common.NewError(err)
	}
	return ok, err
}

func (m *TokenManager) VerifyCpeToken(token string, mac string) (bool, string, int, error) {
	ok, partner, trust, err := m.verifyFn(m.decodeKeys, m.cpeKids, m.cpeCapabilities, token, mac)
	if err != nil {
		return ok, "", trust, common.NewError(err)
	}
	return ok, partner, trust, nil
}

func (m *TokenManager) SetVerifyFunc(fn VerifyFunc) {
	m.verifyFn = fn
}

func (m *TokenManager) ParseCpeToken(tokenStr string) (map[string]string, error) {
	parser := &jwt.Parser{}
	uclaims := jwt.MapClaims{}
	token, _, err := parser.ParseUnverified(tokenStr, uclaims)
	if err != nil {
		return nil, common.NewError(err)
	}

	// check kid
	var kid string
	if itf, ok := token.Header["kid"]; ok {
		kid = itf.(string)
	}
	if len(kid) == 0 {
		return nil, common.NewError(fmt.Errorf("error in reading kid from header"))
	}
	if !util.Contains(m.cpeKids, kid) {
		return nil, common.NewError(fmt.Errorf("token kid=%v, not in validKids=%v", kid, m.cpeKids))
	}
	decodeKey, ok := m.decodeKeys[kid]
	if !ok {
		return nil, common.NewError(fmt.Errorf("key object missing, kid=%v", kid))
	}

	// capabilities check is skipped for now

	claims := jwt.MapClaims{}
	token, err = jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) { return decodeKey, nil })
	if err != nil {
		return nil, common.NewError(err)
	}

	// TODO eval if more claims need to be returned, for now only two
	data := map[string]string{}
	if itf, ok := claims["mac"]; ok {
		mac := itf.(string)
		if len(mac) > 0 {
			data["mac"] = mac
		}
	}
	if itf, ok := claims["partner-id"]; ok {
		partner := itf.(string)
		if len(partner) > 0 {
			data["partner"] = partner
		}
	}

	return data, nil
}

func VerifyToken(decodeKeys map[string]*rsa.PublicKey, validKids []string, requiredCapabilities []string, vargs ...string) (bool, string, int, error) {
	tokenString := vargs[0]
	var kid string
	var trust int

	parser := &jwt.Parser{}

	// this is the claims before the token is verified by the public key
	uclaims := jwt.MapClaims{}
	if token, _, err := parser.ParseUnverified(tokenString, uclaims); err == nil {
		// check kid
		rawkid, ok := token.Header["kid"]
		if !ok {
			return false, "", trust, common.NewError(fmt.Errorf("missing kid in token"))
		}
		kid, ok = rawkid.(string)
		if !ok {
			return false, "", trust, common.NewError(fmt.Errorf("error in reading kid from header"))
		}

		ok = false
		for _, k := range validKids {
			if kid == k {
				ok = true
				break
			}
		}
		if !ok {
			return false, "", trust, common.NewError(fmt.Errorf("token kid=%v, not in validKids=%v", kid, validKids))
		}

		// check capabilities, if requiredCapabilities is nonempty
		if len(requiredCapabilities) > 0 {
			isCapable := false
			if capitfs, ok := uclaims["capabilities"]; ok {
				capvalues, ok1 := capitfs.([]interface{})
				if ok1 {
					for _, capvalue := range capvalues {
						for _, rc := range requiredCapabilities {
							if rc == capvalue {
								isCapable = true
								break
							}
						}
						if isCapable {
							break
						}
					}
				}
			}
			if !isCapable {
				return false, "", trust, common.NewError(fmt.Errorf("token without proper capabilities"))
			}
		}
	} else {
		return false, "", trust, common.NewError(err)
	}

	decodeKey, ok := decodeKeys[kid]
	if !ok {
		return false, "", trust, common.NewError(fmt.Errorf("key object missing, kid=%v", kid))
	}

	claims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) { return decodeKey, nil }); err != nil {
		return false, "", trust, common.NewError(err)
	}

	if len(vargs) > 1 {
		mac := vargs[1]
		// mac must match
		isMatched := false
		if macitf, ok := claims["mac"]; ok {
			if macstr, ok := macitf.(string); ok {
				if strings.ToLower(mac) == strings.ToLower(macstr) {
					isMatched = true
				} else {
					return false, "", trust, common.NewError(fmt.Errorf("mac in token(%v) does not match mac in claims(%v)", mac, macstr))
				}
			}
		}
		if !isMatched {
			return false, "", trust, common.NewError(fmt.Errorf("mac in token(%v) does not match claims=%v", mac, claims))
		}
	}

	// parse partner
	partner := "comcast"
	if itf, ok := claims["partner-id"]; ok {
		partner = itf.(string)
	}

	if itf, ok := claims["trust"]; ok {
		trust = util.ToInt(itf)
	}

	return true, partner, trust, nil
}
