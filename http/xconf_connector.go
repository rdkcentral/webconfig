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
	"net/http"

	owcommon "github.com/rdkcentral/webconfig/common"
	"github.com/go-akka/configuration"
	log "github.com/sirupsen/logrus"
)

const (
	xconfHostDefault = "http://qa2.xconfds.coast.xcal.tv:8080"
	xconfUrlTemplate = "%s/loguploader/getTelemetryProfiles?%s"
)

type XconfConnector struct {
	*HttpClient
	host        string
	serviceName string
}

func NewXconfConnector(conf *configuration.Config, tlsConfig *tls.Config) *XconfConnector {
	serviceName := "xconf"
	confKey := fmt.Sprintf("webconfig.%v.host", serviceName)
	host := conf.GetString(confKey, xconfHostDefault)

	return &XconfConnector{
		// last param indicates no traces to be generated
		HttpClient:  NewHttpClient(conf, serviceName, tlsConfig, false),
		host:        host,
		serviceName: serviceName,
	}
}

func (c *XconfConnector) XconfHost() string {
	return c.host
}

func (c *XconfConnector) SetXconfHost(host string) {
	c.host = host
}

func (c *XconfConnector) ServiceName() string {
	return c.serviceName
}

func (c *XconfConnector) GetProfiles(urlSuffix string, fields log.Fields) ([]byte, http.Header, error) {
	url := fmt.Sprintf(xconfUrlTemplate, c.XconfHost(), urlSuffix)
	rbytes, resHeader, err := c.DoWithRetries("GET", url, nil, nil, fields, c.ServiceName())
	if err != nil {
		return rbytes, resHeader, owcommon.NewError(err)
	}
	return rbytes, resHeader, nil
}
