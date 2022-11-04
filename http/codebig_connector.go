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
package http

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/go-akka/configuration"
	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
)

const (
	defaultCodebigHost        = "https://sat-prod.codebig2.net"
	codebigServiceName        = "codebig"
	codebigUrlTemplate        = "%s/oauth/token"
	codebigPartnerUrlTemplate = "%s/oauth/token?partners=%s"
)

type CodebigConnector struct {
	*HttpClient
	host    string
	headers map[string]string
}

type CodebigResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"score"`
	TokenType    string `json:"token_type"`
	ResponseCode int    `json:"responseCode"`
	Description  string `json:"description"`
}

func NewCodebigConnector(conf *configuration.Config, satClientId, satClientSecret string, tlsConfig *tls.Config) *CodebigConnector {
	confKey := fmt.Sprintf("webconfig.%v.host", codebigServiceName)
	host := conf.GetString(confKey, defaultCodebigHost)
	headers := map[string]string{
		"X-Client-Id":     satClientId,
		"X-Client-Secret": satClientSecret,
	}

	return &CodebigConnector{
		HttpClient: NewHttpClient(conf, codebigServiceName, tlsConfig),
		host:       host,
		headers:    headers,
	}
}

func (c *CodebigConnector) CodebigHost() string {
	return c.host
}

func (c *CodebigConnector) SetCodebigHost(host string) {
	c.host = host
}

func (c *CodebigConnector) GetToken(fields log.Fields, vargs ...string) (string, error) {
	var token string
	var url string

	if len(vargs) > 0 {
		partnerId := vargs[0]
		url = fmt.Sprintf(codebigPartnerUrlTemplate, c.CodebigHost(), partnerId)
	} else {
		url = fmt.Sprintf(codebigUrlTemplate, c.CodebigHost())
	}
	rbytes, _, err := c.DoWithRetries("POST", url, c.headers, nil, fields, codebigServiceName)
	if err != nil {
		return token, common.NewError(err)
	}

	var codebigResponse CodebigResponse
	if err := json.Unmarshal(rbytes, &codebigResponse); err != nil {
		return token, common.NewError(err)
	}

	token = codebigResponse.AccessToken
	if len(token) == 0 {
		err := fmt.Errorf("%v", codebigResponse.Description)
		return token, common.NewError(err)
	}

	return token, nil
}
