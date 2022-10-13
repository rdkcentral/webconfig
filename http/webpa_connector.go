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
	"fmt"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/go-akka/configuration"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	defaultWebpaHost = "https://api.example.com"
	webpaServiceName = "webpa"
	webpaUrlTemplate = "%s/api/%s/device/mac:%s/config"
	satUrlBase       = "https://token.example.com/oauth/token"
	webpaError404    = `{"code": 521, "message": "Device not found in webpa"}`
	webpaError520    = `{"code": 520, "message": "Error unsupported namespace"}`
)

var (
	PokeBody = []byte(`{"parameters":[{"dataType":0,"name":"Device.X_RDK_WebConfig.ForceSync","value":"root"}]}`)
)

type WebpaConnector struct {
	*HttpClient
	host string
}

func NewWebpaConnector(conf *configuration.Config, tlsConfig *tls.Config) *WebpaConnector {
	confKey := fmt.Sprintf("webconfig.%v.host", webpaServiceName)
	host := conf.GetString(confKey, defaultWebpaHost)

	return &WebpaConnector{
		HttpClient: NewHttpClient(conf, webpaServiceName, tlsConfig),
		host:       host,
	}
}

func (c *WebpaConnector) WebpaHost() string {
	return c.host
}

func (c *WebpaConnector) SetWebpaHost(host string) {
	c.host = host
}

func (c *WebpaConnector) Patch(cpeMac string, token string, bbytes []byte, fields log.Fields, apiVersion string) (string, error) {
	url := fmt.Sprintf(webpaUrlTemplate, c.WebpaHost(), apiVersion, cpeMac)

	var traceId string
	if itf, ok := fields["trace_id"]; ok {
		traceId = itf.(string)
	}
	if len(traceId) == 0 {
		traceId = uuid.New().String()
	}

	t := time.Now().UnixNano() / 1000
	transactionId := fmt.Sprintf("%s_____%015X", traceId, t)
	headers := map[string]string{
		"Authorization":          fmt.Sprintf("Bearer %s", token),
		"X-Webpa-Transaction-Id": transactionId,
	}
	_, err := c.DoWithRetries("PATCH", url, headers, bbytes, fields, webpaServiceName)
	if err != nil {
		return transactionId, common.NewError(err)
	}

	return transactionId, nil
}
