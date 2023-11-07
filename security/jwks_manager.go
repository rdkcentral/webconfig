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
	"context"
	"fmt"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"github.com/go-akka/configuration"
	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"
)

const (
	defaultRefreshInterval = 86400
)

type JwksManager struct {
	jwks            *keyfunc.JWKS
	apiCapabilities []string
}

func NewJwksManager(conf *configuration.Config, ctx context.Context) (*JwksManager, error) {
	jwksUrl := conf.GetString("webconfig.jwt.api_token.jwks_url")
	if len(jwksUrl) == 0 {
		err := fmt.Errorf("empty webconfig.jwt.api_token.jwks_url")
		return nil, common.NewError(err)
	}

	refreshInterval := conf.GetInt32("webconfig.jwt.api_token.jwks_refresh_in_secs", defaultRefreshInterval)

	options := keyfunc.Options{
		Ctx:                 ctx,
		RefreshErrorHandler: LogRefreshError,
		RefreshInterval:     time.Duration(refreshInterval) * time.Second,
		RefreshRateLimit:    time.Minute * 5,
		RefreshTimeout:      time.Second * 10,
		RefreshUnknownKID:   true,
	}

	jwks, err := keyfunc.Get(jwksUrl, options)
	if err != nil {
		return nil, common.NewError(err)
	}

	return &JwksManager{
		jwks:            jwks,
		apiCapabilities: conf.GetStringList("webconfig.jwt.api_token.capabilities"),
	}, nil
}

func (m *JwksManager) VerifyApiToken(tokenStr string) (bool, error) {
	token, err := jwt.Parse(tokenStr, m.jwks.Keyfunc)
	if err != nil {
		return false, common.NewError(err)
	}

	// validate against capabilities
	claims := token.Claims
	if mclaims, ok := claims.(jwt.MapClaims); ok {
		if itf, ok := mclaims["capabilities"]; ok {
			if iitfs, ok := itf.([]interface{}); ok {
				for _, iitf := range iitfs {
					ss := iitf.(string)
					if util.Contains(m.apiCapabilities, ss) {
						return true, nil
					}
				}
			}
		}
	}
	return false, common.NoCapabilitiesError{}
}

func LogRefreshError(err error) {
	fields := log.Fields{
		"logger": "codebig",
	}
	message := fmt.Sprintf("There was an error with the jwt.Keyfunc.Get() Error: %s", err.Error())
	log.WithFields(fields).Error(message)
}
