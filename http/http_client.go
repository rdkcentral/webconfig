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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-akka/configuration"
	"github.com/google/uuid"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
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

type StatusHandlerFunc func([]byte) ([]byte, http.Header, bool, error)

type HttpClient struct {
	*http.Client
	retries              int
	retryInMsecs         int
	statusHandlerFuncMap map[int]StatusHandlerFunc
	userAgent            string
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
	userAgent := conf.GetString("webconfig.http_client.user_agent")

	var transport http.RoundTripper = &http.Transport{
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
	}

	return &HttpClient{
		Client: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(readTimeout) * time.Second,
		},
		retries:              retries,
		retryInMsecs:         retryInMsecs,
		statusHandlerFuncMap: map[int]StatusHandlerFunc{},
		userAgent:            userAgent,
	}
}

func (c *HttpClient) Do(method string, url string, header http.Header, bbytes []byte, auditFields log.Fields, loggerName string, retry int) ([]byte, http.Header, bool, error) {
	fields := common.FilterLogFields(auditFields)

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
		return nil, nil, false, common.NewError(fmt.Errorf("method=%v", method))
	}

	if err != nil {
		return nil, nil, true, common.NewError(err)
	}

	if header == nil {
		header = make(http.Header)
	}

	if loggerName == webpaServiceName || loggerName == asyncWebpaServiceName {
		var xmTraceId string
		if itf, ok := fields["xmoney_trace_id"]; ok {
			xmTraceId = itf.(string)
		}
		if len(xmTraceId) == 0 {
			xmTraceId = uuid.New().String()
		}
		t := time.Now().UnixNano() / 1000
		transactionId := fmt.Sprintf("%s_____%015x", xmTraceId, t)
		header.Set("X-Webpa-Transaction-Id", transactionId)
	}

	req.Header = header.Clone()
	if len(c.userAgent) > 0 {
		req.Header.Set(common.HeaderUserAgent, c.userAgent)
	}

	logHeader := header.Clone()
	auth := logHeader.Get("Authorization")
	if len(auth) > 0 {
		logHeader.Set("Authorization", "****")
	}

	var userAgent string
	if itf, ok := fields["user_agent"]; ok {
		if x := itf.(string); len(x) > 0 {
			userAgent = x
		}
	}

	fields["logger"] = loggerName
	fields[fmt.Sprintf("%v_method", loggerName)] = method
	fields[fmt.Sprintf("%v_url", loggerName)] = url
	fields[fmt.Sprintf("%v_headers", loggerName)] = util.HeaderToMap(logHeader)
	bodyKey := fmt.Sprintf("%v_body", loggerName)

	var longBody, longResponse string
	if bbytes != nil && len(bbytes) > 0 {
		bdict := util.Dict{}
		err = json.Unmarshal(bbytes, &bdict)
		if err != nil {
			bodyKey = fmt.Sprintf("%v_body_text", loggerName)
			x := base64.StdEncoding.EncodeToString(bbytes)
			if len(x) < 1000 {
				fields[bodyKey] = x
			} else {
				fields[bodyKey] = "long body len " + strconv.Itoa(len(x))
				longBody = x
			}
		} else {
			fields[bodyKey] = bdict
		}
	}

	var startMessage string
	if retry > 0 {
		startMessage = fmt.Sprintf("%v retry=%v starts", loggerName, retry)
	} else {
		startMessage = fmt.Sprintf("%v starts", loggerName)
	}

	if userAgent != "mget" {
		log.WithFields(fields).Info(startMessage)
		if len(longBody) > 0 {
			tfields := common.FilterLogFields(fields)
			tfields[bodyKey] = longBody
			log.WithFields(tfields).Trace(startMessage)
		}
	}

	startTime := time.Now()

	// the core http call
	res, err := c.Client.Do(req)
	// err should be *url.Error

	tdiff := time.Since(startTime)
	duration := tdiff.Milliseconds()
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
		if userAgent != "mget" {
			log.WithFields(fields).Info(endMessage)
		}
		if ue, ok := err.(*neturl.Error); ok {
			innerErr := ue.Err
			if ue.Timeout() {
				rherr := common.RemoteHttpError{
					Message:    ue.Error(),
					StatusCode: http.StatusGatewayTimeout,
				}
				return nil, nil, true, common.NewError(rherr)
			}
			if errors.Is(innerErr, io.EOF) {
				rherr := common.RemoteHttpError{
					Message:    ue.Error(),
					StatusCode: http.StatusBadGateway,
				}
				return nil, nil, true, common.NewError(rherr)
			}
			if _, ok := innerErr.(*net.OpError); ok {
				rherr := common.RemoteHttpError{
					Message:    ue.Error(),
					StatusCode: http.StatusServiceUnavailable,
				}
				return nil, nil, true, common.NewError(rherr)
			}
			// Unknown err still appear as 500
		}
		return nil, nil, true, common.NewError(err)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	fields[fmt.Sprintf("%v_status", loggerName)] = res.StatusCode
	rbytes, err := io.ReadAll(res.Body)
	if err != nil {
		// XPC-23206 catch the timeout/context_cancellation error
		// ex: context deadline exceeded (Client.Timeout or context cancellation while reading body)
		lowerErrText := strings.ToLower(err.Error())
		if strings.Contains(lowerErrText, "timeout") {
			err = common.RemoteHttpError{
				Message:    err.Error(),
				StatusCode: http.StatusGatewayTimeout,
			}
		}
		fields[errorKey] = err.Error()
		if userAgent != "mget" {
			log.WithFields(fields).Info(endMessage)
		}
		return nil, nil, true, common.NewError(err)
	}

	rbody := string(rbytes)
	resp := util.Dict{}
	err = json.Unmarshal(rbytes, &resp)
	responseKey := fmt.Sprintf("%v_response_text", loggerName)

	if err != nil {
		if loggerName == "mqtt" && (res.StatusCode == 404 || res.StatusCode == 202) {
			fields[responseKey] = strings.TrimSpace(rbody)
		} else {
			x := base64.StdEncoding.EncodeToString(rbytes)
			if len(x) < 1000 {
				fields[responseKey] = x
			} else {
				fields[responseKey] = "long response len " + strconv.Itoa(len(x))
				longResponse = x
			}
		}
	} else {
		fields[fmt.Sprintf("%v_response", loggerName)] = resp
	}
	if userAgent != "mget" {
		log.WithFields(fields).Info(endMessage)
		if len(longResponse) > 0 {
			tfields := common.FilterLogFields(fields)
			tfields[responseKey] = longResponse
			log.WithFields(tfields).Trace(endMessage)
		}
	}

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
			return rbytes, nil, false, common.NewError(err)
		}
		return rbytes, nil, true, common.NewError(err)
	} else if res.StatusCode > 200 {
		var pokeResponse PokeResponse
		var message string
		if err := json.Unmarshal(rbytes, &pokeResponse); err == nil {
			if len(pokeResponse.Parameters) > 0 {
				message = pokeResponse.Parameters[0].Message
			}
		}
		if len(message) == 0 {
			message = http.StatusText(res.StatusCode)
		}
		rherr := common.RemoteHttpError{
			Message:    message,
			StatusCode: res.StatusCode,
		}
		return rbytes, nil, false, common.NewError(rherr)
	}
	return rbytes, res.Header, false, nil
}

func (c *HttpClient) DoWithRetries(method string, url string, rHeader http.Header, bbytes []byte, fields log.Fields, loggerName string) ([]byte, http.Header, error) {
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
		respBytes, respHeader, cont, err = c.Do(method, url, rHeader, cbytes, fields, loggerName, i)
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
