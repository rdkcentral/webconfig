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

	"github.com/go-akka/configuration"
	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
)

const (
	mqttHostDefault = "https://hcbroker.staging.us-west-2.plume.comcast.net"
	mqttUrlTemplate = "%s/v2/mqttpub/x/to/%s"
)

type MqttConnector struct {
	*HttpClient
	host        string
	serviceName string
}

func NewMqttConnector(conf *configuration.Config, tlsConfig *tls.Config) *MqttConnector {
	serviceName := "mqtt"
	confKey := fmt.Sprintf("webconfig.%v.host", serviceName)
	host := conf.GetString(confKey, mqttHostDefault)

	return &MqttConnector{
		HttpClient:  NewHttpClient(conf, serviceName, tlsConfig),
		host:        host,
		serviceName: serviceName,
	}
}

func (c *MqttConnector) MqttHost() string {
	return c.host
}

func (c *MqttConnector) SetMqttHost(host string) {
	c.host = host
}

func (c *MqttConnector) ServiceName() string {
	return c.serviceName
}

func (c *MqttConnector) PostMqtt(cpeMac string, bbytes []byte, fields log.Fields) ([]byte, error) {
	url := fmt.Sprintf(mqttUrlTemplate, c.MqttHost(), cpeMac)
	rbytes, _, err := c.DoWithRetries("POST", url, nil, bbytes, fields, c.ServiceName())
	if err != nil {
		return rbytes, common.NewError(err)
	}
	return rbytes, nil
}
