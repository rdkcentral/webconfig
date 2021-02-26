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
	"fmt"
	"strings"

	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/rdkcentral/webconfig/common"
)

func VerifyBySermo(decodeKeys map[string]*rsa.PublicKey, validKids []string, requiredCapabilities []string, vargs ...string) (bool, error) {
	tokenString := vargs[0]
	kid, err := ParseKidFromTokenHeader(tokenString)
	if err != nil {
		return false, common.NewError(err)
	}

	token, err := jws.ParseJWT([]byte(tokenString))
	if err != nil {
		return false, common.NewError(err)
	}
	claims := token.Claims()

	// check kid is valid
	ok := false
	for _, k := range validKids {
		if kid == k {
			ok = true
			break
		}
	}
	if !ok {
		return false, common.NewError(fmt.Errorf("token kid=%v, not in validKids=%v", kid, validKids))
	}

	// check capabilities, if requiredCapabilities is nonempty
	if len(requiredCapabilities) > 0 {
		isCapable := false
		if capitfs, ok := claims["capabilities"]; ok {
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
			return false, common.NewError(fmt.Errorf("token without proper capabilities"))
		}
	}

	// verify mac
	if len(vargs) > 1 {
		mac := vargs[1]
		// mac must match
		isMatched := false
		if macitf, ok := claims["mac"]; ok {
			if macstr, ok := macitf.(string); ok {
				if strings.ToLower(mac) == strings.ToLower(macstr) {
					isMatched = true
				} else {
					return false, common.NewError(fmt.Errorf("mac in token(%v) does not match mac in claims(%v)", mac, macstr))
				}
			}
		}
		if !isMatched {
			return false, common.NewError(fmt.Errorf("mac in token(%v) does not match claims=%v", mac, claims))
		}
	}

	decodeKey, ok := decodeKeys[kid]
	if !ok {
		return false, common.NewError(fmt.Errorf("key object missing, kid=%v", kid))
	}

	// validate
	if err := token.Validate(decodeKey, crypto.SigningMethodRS256); err != nil {
		return false, common.NewError(err)
	}

	return true, nil
}
