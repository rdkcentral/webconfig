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
	"github.com/rdkcentral/webconfig/common"
	owcommon "github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"
)

const (
	defaultUpstreamHost        = "http://localhost:12348"
	defaultUpstreamUrlTemplate = "%s/%s"
	defaultProfileUrlTemplate  = "%s/%s/%s"
)

type UpstreamConnector struct {
	*HttpClient
	host                string
	serviceName         string
	upstreamUrlTemplate string
	profileUrlTemplate  string
}

func NewUpstreamConnector(conf *configuration.Config, tlsConfig *tls.Config) *UpstreamConnector {
	serviceName := "upstream"
	host := conf.GetString("webconfig.upstream.host", defaultUpstreamHost)
	upstreamUrlTemplate := conf.GetString("webconfig.upstream.url_template", defaultUpstreamUrlTemplate)
	profileUrlTemplate := conf.GetString("webconfig.upstream.profile_url_template", defaultProfileUrlTemplate)

	return &UpstreamConnector{
		HttpClient:          NewHttpClient(conf, serviceName, tlsConfig),
		host:                host,
		serviceName:         serviceName,
		upstreamUrlTemplate: upstreamUrlTemplate,
		profileUrlTemplate:  profileUrlTemplate,
	}
}

func (c *UpstreamConnector) UpstreamHost() string {
	return c.host
}

func (c *UpstreamConnector) SetUpstreamHost(host string) {
	c.host = host
}

func (c *UpstreamConnector) ServiceName() string {
	return c.serviceName
}

func (c *UpstreamConnector) PostUpstream(mac string, header http.Header, bbytes []byte, fields log.Fields) ([]byte, http.Header, error) {
	url := fmt.Sprintf(c.upstreamUrlTemplate, c.UpstreamHost(), mac)

	if itf, ok := fields["audit_id"]; ok {
		auditId := itf.(string)
		if len(auditId) > 0 {
			header.Set(common.HeaderAuditid, auditId)
		}
	}

	if itf, ok := fields["app_name"]; ok {
		appName := itf.(string)
		if len(appName) > 0 {
			header.Set(common.HeaderSourceAppName, appName)
		}
	}

	rbytes, header, err := c.DoWithRetries("POST", url, header, bbytes, fields, c.ServiceName())
	if err != nil {
		return rbytes, header, owcommon.NewError(err)
	}
	return rbytes, header, nil
}

func (c *UpstreamConnector) GetUpstreamProfiles(mac, queryParams string, header http.Header, fields log.Fields) ([]byte, http.Header, error) {
	url := fmt.Sprintf(c.profileUrlTemplate, c.UpstreamHost(), mac, queryParams)

	if itf, ok := fields["audit_id"]; ok {
		auditId := itf.(string)
		if len(auditId) > 0 {
			header.Set(common.HeaderAuditid, auditId)
		}
	}

	if itf, ok := fields["app_name"]; ok {
		appName := itf.(string)
		if len(appName) > 0 {
			header.Set(common.HeaderSourceAppName, appName)
		}
	}

	rbytes, header, err := c.DoWithRetries("GET", url, header, nil, fields, c.ServiceName())
	if err != nil {
		return rbytes, header, owcommon.NewError(err)
	}
	return rbytes, header, nil
}
