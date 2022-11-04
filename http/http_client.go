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
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/go-akka/configuration"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
)

const (
	defaultConnectTimeout      = 30
	defaultReadTimeout         = 30
	defaultMaxIdleConnsPerHost = 100
	defaultKeepaliveTimeout    = 30
	defaultRetries             = 3
	defaultRetriesInMsecs      = 1000
)

type ErrorResponse struct {
	Message string `json:"message"`
}

type StatusHandlerFunc func([]byte) ([]byte, http.Header, error, bool)

type HttpClient struct {
	*http.Client
	retries              int
	retryInMsecs         int
	statusHandlerFuncMap map[int]StatusHandlerFunc
}

func NewHttpClient(conf *configuration.Config, serviceName string, tlsConfig *tls.Config) *HttpClient {
	confKey := fmt.Sprintf("webconfig.%v.connect_timeout_in_secs", serviceName)
	connectTimeout := int(conf.GetInt32(confKey, defaultConnectTimeout))

	confKey = fmt.Sprintf("webconfig.%v.read_timeout_in_secs", serviceName)
	readTimeout := int(conf.GetInt32(confKey, defaultReadTimeout))

	confKey = fmt.Sprintf("webconfig.%v.max_idle_conns_per_host", serviceName)
	maxIdleConnsPerHost := int(conf.GetInt32(confKey, defaultMaxIdleConnsPerHost))

	confKey = fmt.Sprintf("webconfig.%v.keepalive_timeout_in_secs", serviceName)
	keepaliveTimeout := int(conf.GetInt32(confKey, defaultKeepaliveTimeout))

	confKey = fmt.Sprintf("webconfig.%v.retries", serviceName)
	retries := int(conf.GetInt32(confKey, defaultRetries))

	confKey = fmt.Sprintf("webconfig.%v.retry_in_msecs", serviceName)
	retryInMsecs := int(conf.GetInt32(confKey, defaultRetriesInMsecs))

	return &HttpClient{
		Client: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   time.Duration(connectTimeout) * time.Second,
					KeepAlive: time.Duration(keepaliveTimeout) * time.Second,
				}).DialContext,
				MaxIdleConns:          0,
				MaxIdleConnsPerHost:   maxIdleConnsPerHost,
				IdleConnTimeout:       time.Duration(keepaliveTimeout) * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig:       tlsConfig,
			},
			Timeout: time.Duration(readTimeout) * time.Second,
		},
		retries:              retries,
		retryInMsecs:         retryInMsecs,
		statusHandlerFuncMap: map[int]StatusHandlerFunc{},
	}
}

func (c *HttpClient) Do(method string, url string, headerMap map[string]string, bbytes []byte, baseFields log.Fields, loggerName string, retry int) ([]byte, http.Header, error, bool) {
	// verify a response is received
	var req *http.Request
	var err error
	switch method {
	case "GET":
		req, err = http.NewRequest(method, url, nil)
	case "POST", "PATCH":
		req, err = http.NewRequest(method, url, bytes.NewReader(bbytes))
	case "DELETE":
		req, err = http.NewRequest(method, url, nil)
	default:
		return nil, nil, common.NewError(fmt.Errorf("method=%v", method)), false
	}

	if err != nil {
		return nil, nil, common.NewError(err), true
	}

	logHeaders := map[string]string{}
	for k, v := range headerMap {
		req.Header.Set(k, v)
		if k == "Authorization" {
			logHeaders[k] = "****"
		} else {
			logHeaders[k] = v
		}
	}

	tfields := util.CopyLogFields(baseFields)
	tfields["logger"] = loggerName
	tfields[fmt.Sprintf("%v_method", loggerName)] = method
	tfields[fmt.Sprintf("%v_url", loggerName)] = url
	tfields[fmt.Sprintf("%v_headers", loggerName)] = logHeaders
	bodyKey := fmt.Sprintf("%v_body", loggerName)
	if bbytes != nil && len(bbytes) > 0 {
		bdict := util.Dict{}
		err = json.Unmarshal(bbytes, &bdict)
		if err != nil {
			bodyKey = fmt.Sprintf("%v_body_text", loggerName)
			tfields[bodyKey] = base64.StdEncoding.EncodeToString(bbytes)
		} else {
			tfields[bodyKey] = bdict
		}
	}
	fields := util.CopyLogFields(tfields)

	var startMessage string
	if retry > 0 {
		startMessage = fmt.Sprintf("%v retry=%v starts", loggerName, retry)
	} else {
		startMessage = fmt.Sprintf("%v starts", loggerName)
	}
	log.WithFields(fields).Info(startMessage)
	startTime := time.Now()

	res, err := c.Client.Do(req)

	tdiff := time.Now().Sub(startTime)
	duration := tdiff.Nanoseconds() / 1000000
	fields[fmt.Sprintf("%v_duration", loggerName)] = duration

	delete(fields, bodyKey)

	var endMessage string
	if retry > 0 {
		endMessage = fmt.Sprintf("%v retry=%v ends", loggerName, retry)
	} else {
		endMessage = fmt.Sprintf("%v ends", loggerName)
	}

	errorKey := fmt.Sprintf("%v_error", loggerName)

	if err != nil {
		fields[errorKey] = err.Error()
		log.WithFields(fields).Info(endMessage)
		return nil, nil, common.NewError(err), true
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	fields[fmt.Sprintf("%v_status", loggerName)] = res.StatusCode
	rbytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fields[errorKey] = err.Error()
		log.WithFields(fields).Info(endMessage)
		return nil, nil, common.NewError(err), false
	}

	rbody := string(rbytes)
	// XPC-13444
	resp := util.Dict{}
	err = json.Unmarshal(rbytes, &resp)
	if err != nil {
		fields[fmt.Sprintf("%v_response_text", loggerName)] = rbody
	} else {
		fields[fmt.Sprintf("%v_response", loggerName)] = resp
	}
	log.WithFields(fields).Info(fmt.Sprintf("%v ends", loggerName))

	// check if there is any customized statusHandler
	if fn := c.StatusHandler(res.StatusCode); fn != nil {
		return fn(rbytes)
	}

	if res.StatusCode >= 400 {
		var errorMessage string
		if len(rbody) > 0 && len(resp) > 0 {
			errorMessage = resp.GetString("message")

			if len(errorMessage) == 0 {
				errorMessage = rbody
			}
		} else {
			errorMessage = http.StatusText(res.StatusCode)
		}
		err := common.RemoteHttpError{
			Message:    errorMessage,
			StatusCode: res.StatusCode,
		}

		switch res.StatusCode {
		case http.StatusForbidden, http.StatusBadRequest, http.StatusNotFound:
			return rbytes, nil, common.NewError(err), false
		}
		return rbytes, nil, common.NewError(err), true
	}
	return rbytes, res.Header, nil, false
}

func (c *HttpClient) DoWithRetries(method string, url string, inHeaders map[string]string, bbytes []byte, fields log.Fields, loggerName string) ([]byte, http.Header, error) {
	var traceId string
	if itf, ok := fields["trace_id"]; ok {
		traceId = itf.(string)
	}
	if len(traceId) == 0 {
		traceId = uuid.New().String()
	}

	xmoney := fmt.Sprintf("trace-id=%s;parent-id=0;span-id=0;span-name=%s", traceId, loggerName)
	headers := map[string]string{
		"X-Moneytrace": xmoney,
	}
	if inHeaders != nil {
		for k, v := range inHeaders {
			headers[k] = v
		}
	}

	// var res *http.Response
	var respBytes []byte
	var respHeader http.Header
	var err error
	var cont bool

	i := 0
	// i=0 is NOT considered a retry, so it ends at i=c.webpaRetries
	for i = 0; i <= c.retries; i++ {
		cbytes := make([]byte, len(bbytes))
		copy(cbytes, bbytes)
		if i > 0 {
			time.Sleep(time.Duration(c.retryInMsecs) * time.Millisecond)
		}
		respBytes, respHeader, err, cont = c.Do(method, url, headers, cbytes, fields, loggerName, i)
		if !cont {
			break
		}
	}

	if err != nil {
		return respBytes, respHeader, common.NewError(err)
	}
	return respBytes, respHeader, nil
}

func (c *HttpClient) SetStatusHandler(status int, fn StatusHandlerFunc) {
	c.statusHandlerFuncMap[status] = fn
}

func (c *HttpClient) StatusHandler(status int) StatusHandlerFunc {
	if fn, ok := c.statusHandlerFuncMap[status]; ok {
		return fn
	}
	return nil
}
