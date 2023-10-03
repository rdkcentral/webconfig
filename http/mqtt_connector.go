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
	"time"

	"github.com/go-akka/configuration"
	"github.com/google/uuid"
	"github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"
)

const (
	mqttHostDefault = "https://hcbroker.staging.us-west-2.plume.comcast.net"
	mqttUrlTemplate = "%s/v2/mqtt/pub/x/to/%s/webconfig"
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

	var traceId, xmTraceId, outTraceparent, outTracestate string
	if itf, ok := fields["xmoney_trace_id"]; ok {
		xmTraceId = itf.(string)
	}
	if len(xmTraceId) == 0 {
		xmTraceId = uuid.New().String()
	}

	if len(traceId) == 0 {
		traceId = xmTraceId
	}
	if itf, ok := fields["out_traceparent"]; ok {
		outTraceparent = itf.(string)
	}
	if itf, ok := fields["out_tracestate"]; ok {
		outTracestate = itf.(string)
	}

	t := time.Now().UnixNano() / 1000
	transactionId := fmt.Sprintf("%s_____%015x", xmTraceId, t)
	xmoney := fmt.Sprintf("trace-id=%s;parent-id=0;span-id=0;span-name=%s", xmTraceId, c.ServiceName())
	header := make(http.Header)
	header.Set("X-Webpa-Transaction-Id", transactionId)
	header.Set("X-Moneytrace", xmoney)
	header.Set(common.HeaderTraceparent, outTraceparent)
	header.Set(common.HeaderTracestate, outTracestate)

	rbytes, _, err := c.DoWithRetries("POST", url, header, bbytes, fields, c.ServiceName())
	if err != nil {
		return rbytes, common.NewError(err)
	}
	return rbytes, nil
}
