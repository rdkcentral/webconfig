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
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-akka/configuration"
	"github.com/google/uuid"
	"github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"
)

const (
	defaultWebpaHost      = "http://localhost:12345"
	defaultApiVersion     = "v2"
	webpaServiceName      = "webpa"
	asyncWebpaServiceName = "asyncwebpa"

	defaultWebpaUrlTemplate = "%s/%s/%s"
	webpaError404           = `{"code": 521, "message": "Device not found in webpa"}`
	webpaError520           = `{"code": 520, "message": "Error unsupported namespace"}`

	// a new error code to indicate it is webpa 520
	// but it is caused by some temporary conditions,
	// NOT because webconfig is unavailable
	webpa520NewStatusCode = 524
)

var (
	PokeBody = []byte(`{"parameters":[{"dataType":0,"name":"Device.X_RDK_WebConfig.ForceSync","value":"root"}]}`)
)

type PokeResponseEntry struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type PokeResponse struct {
	Parameters []PokeResponseEntry `json:"parameters"`
	StatusCode int                 `json:"statusCode"`
}

type WebpaConnector struct {
	syncClient       *HttpClient
	asyncClient      *HttpClient
	host             string
	urlTemplate      string
	queue            chan struct{}
	retries          int
	retryInMsecs     int
	asyncPokeEnabled bool
	apiVersion       string
}

func syncHandle520(rbytes []byte) ([]byte, http.Header, bool, error) {
	rerr := common.RemoteHttpError{
		Message:    string(rbytes),
		StatusCode: 520,
	}

	var pres PokeResponse
	if err := json.Unmarshal(rbytes, &pres); err == nil {
		if len(pres.Parameters) > 0 {
			if pres.Parameters[0].Message == "Error unsupported namespace" || pres.Parameters[0].Message == "Request rejected" {
				return rbytes, nil, false, common.NewError(rerr)
			}
		}
	}
	rerr.StatusCode = webpa520NewStatusCode

	return rbytes, nil, false, common.NewError(rerr)
}

func asyncHandle520(rbytes []byte) ([]byte, http.Header, bool, error) {
	rerr := common.RemoteHttpError{
		Message:    string(rbytes),
		StatusCode: 520,
	}

	var pres PokeResponse
	if err := json.Unmarshal(rbytes, &pres); err == nil {
		if len(pres.Parameters) > 0 {
			if pres.Parameters[0].Message == "Error unsupported namespace" || pres.Parameters[0].Message == "Request rejected" {
				return rbytes, nil, false, common.NewError(rerr)
			}
		}
	}
	rerr.StatusCode = webpa520NewStatusCode

	return rbytes, nil, true, common.NewError(rerr)
}

func NewWebpaConnector(conf *configuration.Config, tlsConfig *tls.Config) *WebpaConnector {
	host := conf.GetString("webconfig.webpa.host", defaultWebpaHost)
	asyncPokeEnabled := conf.GetBoolean("webconfig.webpa.async_poke_enabled", false)
	concurrentCalls := int(conf.GetInt32("webconfig.webpa.async_poke_concurrent_calls", 0))
	var queue chan struct{}
	if concurrentCalls > 0 {
		queue = make(chan struct{}, concurrentCalls)
	}

	retries := int(conf.GetInt32("webconfig.webpa.retries", defaultRetries))
	retryInMsecs := int(conf.GetInt32("webconfig.webpa.retry_in_msecs", defaultRetriesInMsecs))
	apiVersion := conf.GetString("webconfig.webpa.api_version", defaultApiVersion)
	urlTemplate := conf.GetString("webconfig.webpa.url_template", defaultWebpaUrlTemplate)

	syncClient := NewHttpClient(conf, webpaServiceName, tlsConfig)
	syncClient.SetStatusHandler(520, syncHandle520)
	asyncClient := NewHttpClient(conf, asyncWebpaServiceName, tlsConfig)
	asyncClient.SetStatusHandler(520, asyncHandle520)

	connector := WebpaConnector{
		syncClient:       syncClient,
		asyncClient:      asyncClient,
		host:             host,
		queue:            queue,
		retries:          retries,
		retryInMsecs:     retryInMsecs,
		asyncPokeEnabled: asyncPokeEnabled,
		apiVersion:       apiVersion,
		urlTemplate:      urlTemplate,
	}

	return &connector
}

func (c *WebpaConnector) WebpaHost() string {
	return c.host
}

func (c *WebpaConnector) SetWebpaHost(host string) {
	c.host = host
}

func (c *WebpaConnector) WebpaUrlTemplate() string {
	return c.urlTemplate
}

func (c *WebpaConnector) SetWebpaUrlTemplate(x string) {
	c.urlTemplate = x
}

func (c *WebpaConnector) ApiVersion() string {
	return c.apiVersion
}

func (c *WebpaConnector) SetApiVersion(apiVersion string) {
	c.apiVersion = apiVersion
}

func (c *WebpaConnector) NewQueue(capacity int) error {
	if c.queue != nil {
		err := fmt.Errorf("queue is already initialized")
		return common.NewError(err)
	}
	c.queue = make(chan struct{}, capacity)
	return nil
}

func (c *WebpaConnector) AsyncPokeEnabled() bool {
	return c.asyncPokeEnabled
}

func (c *WebpaConnector) SetAsyncPokeEnabled(enabled bool) {
	c.asyncPokeEnabled = enabled
}

func (c *WebpaConnector) Patch(rHeader http.Header, cpeMac string, token string, bbytes []byte, fields log.Fields) (string, error) {
	url := fmt.Sprintf(c.WebpaUrlTemplate(), c.WebpaHost(), c.ApiVersion(), cpeMac)

	var traceId, xmTraceId string
	if itf, ok := fields["trace_id"]; ok {
		traceId = itf.(string)
	}

	if itf, ok := fields["xmoney_trace_id"]; ok {
		xmTraceId = itf.(string)
	}
	if len(xmTraceId) == 0 {
		xmTraceId = uuid.New().String()
	}

	if len(traceId) == 0 {
		traceId = xmTraceId
	}

	t := time.Now().UnixNano() / 1000
	transactionId := fmt.Sprintf("%s_____%015x", xmTraceId, t)
	header := rHeader.Clone()
	header.Set("X-Webpa-Transaction-Id", transactionId)

	method := "PATCH"
	_, _, cont, err := c.syncClient.Do(method, url, header, bbytes, fields, webpaServiceName, 0)
	if err != nil {
		var rherr common.RemoteHttpError
		if errors.As(err, &rherr) {
			if rherr.StatusCode == 524 {
				if c.asyncPokeEnabled {
					c.queue <- struct{}{}
					go c.AsyncDoWithRetries(method, url, header, bbytes, fields, asyncWebpaServiceName)
				} else {
					_, err := c.SyncDoWithRetries(method, url, header, bbytes, fields, webpaServiceName)
					if err != nil {
						return transactionId, common.NewError(err)
					}
				}
				return transactionId, common.NewError(err)
			}
		}
		if cont {
			_, _, err := c.syncClient.DoWithRetries("PATCH", url, header, bbytes, fields, webpaServiceName)
			if err != nil {
				return transactionId, common.NewError(err)
			}
			return transactionId, nil
		}
		return transactionId, common.NewError(err)
	}
	return transactionId, nil
}

func (c *WebpaConnector) AsyncDoWithRetries(method string, url string, header http.Header, bbytes []byte, fields log.Fields, loggerName string) {
	tfields := common.FilterLogFields(fields, "status")
	tfields["logger"] = "asyncwebpa"
	for i := 1; i <= c.retries; i++ {
		cbytes := make([]byte, len(bbytes))
		copy(cbytes, bbytes)
		if i > 0 {
			time.Sleep(time.Duration(c.retryInMsecs) * time.Millisecond)
		}
		_, _, cont, _ := c.asyncClient.Do(method, url, header, cbytes, fields, loggerName, i)
		if !cont {
			msg := fmt.Sprintf("finished success after 1 retry")
			if i > 1 {
				fmt.Sprintf("finished success after %v retries", i)
			}
			log.WithFields(tfields).Info(msg)
			break
		}
		if i == c.retries {
			log.WithFields(tfields).Infof("finished failure after %v retries", i)
		}
	}
	<-c.queue
}

// this has 1 less retries compared to the standard DoWithRetries()
func (c *WebpaConnector) SyncDoWithRetries(method string, url string, header http.Header, bbytes []byte, fields log.Fields, loggerName string) ([]byte, error) {
	var rbytes []byte
	var err error
	var cont bool

	for i := 1; i <= c.retries; i++ {
		cbytes := make([]byte, len(bbytes))
		copy(cbytes, bbytes)
		if i > 0 {
			time.Sleep(time.Duration(c.retryInMsecs) * time.Millisecond)
		}
		rbytes, _, cont, err = c.syncClient.Do(method, url, header, cbytes, fields, loggerName, i)
		if !cont {
			// in the case of 524/in-progress, we continue
			var rherr common.RemoteHttpError
			if errors.As(err, &rherr) {
				if rherr.StatusCode == 524 {
					continue
				}
			}
			break
		}
	}
	if err != nil {
		return rbytes, common.NewError(err)
	}
	return rbytes, nil
}
