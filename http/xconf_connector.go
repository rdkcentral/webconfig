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

	"github.com/go-akka/configuration"
	owcommon "github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"
)

const (
	defaultXconfHost        = "http://localhost:12346"
	defaultXconfUrlTemplate = "%s/%s"
)

type XconfConnector struct {
	*HttpClient
	host        string
	serviceName string
	urlTemplate string
}

func NewXconfConnector(conf *configuration.Config, tlsConfig *tls.Config) *XconfConnector {
	serviceName := "xconf"
	host := conf.GetString("webconfig.xconf.host", defaultXconfHost)
	urlTemplate := conf.GetString("webconfig.xconf.url_template", defaultXconfUrlTemplate)

	return &XconfConnector{
		HttpClient:  NewHttpClient(conf, serviceName, tlsConfig),
		host:        host,
		serviceName: serviceName,
		urlTemplate: urlTemplate,
	}
}

func (c *XconfConnector) XconfHost() string {
	return c.host
}

func (c *XconfConnector) SetXconfHost(host string) {
	c.host = host
}

func (c *XconfConnector) XconfUrlTemplate() string {
	return c.urlTemplate
}

func (c *XconfConnector) SetXconfUrlTemplate(x string) {
	c.urlTemplate = x
}

func (c *XconfConnector) ServiceName() string {
	return c.serviceName
}

func (c *XconfConnector) GetProfiles(urlSuffix string, fields log.Fields) ([]byte, http.Header, error) {
	url := fmt.Sprintf(c.XconfUrlTemplate(), c.XconfHost(), urlSuffix)
	rbytes, resHeader, err := c.DoWithRetries("GET", url, nil, nil, fields, c.ServiceName())
	if err != nil {
		return rbytes, resHeader, owcommon.NewError(err)
	}
	return rbytes, resHeader, nil
}
