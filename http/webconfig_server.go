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
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/security"
	"github.com/rdkcentral/webconfig/util"
	"github.com/go-akka/configuration"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	LevelWarn = iota
	LevelInfo
	LevelDebug
	MetricsEnabledDefault            = true
	FactoryResetEnabledDefault       = false
	serverApiTokenAuthEnabledDefault = false
	deviceApiTokenAuthEnabledDefault = true
	tokenApiEnabledDefault           = false
)

type WebconfigServer struct {
	*http.Server
	db.DatabaseClient
	*security.TokenManager
	*common.ServerConfig
	*WebpaConnector
	*CodebigConnector
	tlsConfig                 *tls.Config
	notLoggedHeaders          []string
	metricsEnabled            bool
	factoryResetEnabled       bool
	serverApiTokenAuthEnabled bool
	deviceApiTokenAuthEnabled bool
	tokenApiEnabled           bool
}

func NewTlsConfig(conf *configuration.Config) (*tls.Config, error) {
	certFile := conf.GetString("webconfig.http_client.cert_file")
	if len(certFile) == 0 {
		err := fmt.Errorf("missing file %v\n", certFile)
		return nil, common.NewError(err)
	}
	privateKeyFile := conf.GetString("webconfig.http_client.private_key_file")
	if len(privateKeyFile) == 0 {
		err := fmt.Errorf("missing file %v\n", privateKeyFile)
		return nil, common.NewError(err)
	}
	cert, err := tls.LoadX509KeyPair(certFile, privateKeyFile)
	if err != nil {
		return nil, common.NewError(err)
	}

	return &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}, nil
}

// testOnly=true ==> running unit test
func NewWebconfigServer(sc *common.ServerConfig, testOnly bool, dc db.DatabaseClient) *WebconfigServer {
	conf := sc.Config
	var dbclient db.DatabaseClient
	var err error

	if dc == nil {
		if conf.GetBoolean("webconfig.database.sqlite3.enabled", false) {
			dbclient, err = db.NewSqliteClient(conf, testOnly)
			if err != nil {
				panic(err)
			}
			err = dbclient.SetUp()
			if err != nil {
				panic(err)
			}
		}
	} else {
		dbclient = dc
	}

	metricsEnabled := conf.GetBoolean("webconfig.server.metrics_enabled", MetricsEnabledDefault)
	factoryResetEnabled := conf.GetBoolean("webconfig.server.factory_reset_enabled", FactoryResetEnabledDefault)

	// configure headers that should not be logged
	ignoredHeaders := conf.GetStringList("webconfig.log.ignored_headers")
	ignoredHeaders = append(common.DefaultIgnoredHeaders, ignoredHeaders...)
	var notLoggedHeaders []string
	for _, x := range ignoredHeaders {
		notLoggedHeaders = append(notLoggedHeaders, strings.ToLower(x))
	}

	// tlsConfig, here we ignore any error
	tlsConfig, _ := NewTlsConfig(conf)

	// load codebig credentials
	satClientId := os.Getenv("SAT_CLIENT_ID")
	if len(satClientId) == 0 {
		panic("No env SAT_CLIENT_ID")
	}

	satClientSecret := os.Getenv("SAT_CLIENT_SECRET")
	if len(satClientSecret) == 0 {
		panic("No env SAT_CLIENT_SECRET")
	}

	serverApiTokenAuthEnabled := conf.GetBoolean("webconfig.jwt.server_api_token_auth.enabled", serverApiTokenAuthEnabledDefault)
	deviceApiTokenAuthEnabled := conf.GetBoolean("webconfig.jwt.device_api_token_auth.enabled", deviceApiTokenAuthEnabledDefault)
	tokenApiEnabled := conf.GetBoolean("webconfig.token_api_enabled", tokenApiEnabledDefault)

	return &WebconfigServer{
		Server: &http.Server{
			Addr:         fmt.Sprintf(":%s", conf.GetString("webconfig.server.port")),
			ReadTimeout:  time.Duration(conf.GetInt32("webconfig.server.read_timeout_in_secs", 3)) * time.Second,
			WriteTimeout: time.Duration(conf.GetInt32("webconfig.server.write_timeout_in_secs", 3)) * time.Second,
		},
		DatabaseClient:            dbclient,
		TokenManager:              security.NewTokenManager(conf),
		ServerConfig:              sc,
		WebpaConnector:            NewWebpaConnector(conf, tlsConfig),
		CodebigConnector:          NewCodebigConnector(conf, satClientId, satClientSecret, tlsConfig),
		tlsConfig:                 tlsConfig,
		notLoggedHeaders:          notLoggedHeaders,
		metricsEnabled:            metricsEnabled,
		factoryResetEnabled:       factoryResetEnabled,
		serverApiTokenAuthEnabled: serverApiTokenAuthEnabled,
		deviceApiTokenAuthEnabled: deviceApiTokenAuthEnabled,
		tokenApiEnabled:           tokenApiEnabled,
	}
}

func (s *WebconfigServer) TestingMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xp := NewXpcResponseWriter(w)
		xw := *xp

		if r.Method == "POST" {
			if r.Body != nil {
				if rbytes, err := ioutil.ReadAll(r.Body); err == nil {
					xw.SetBody(string(rbytes))
				}
			} else {
				xw.SetBody("")
			}
		}
		next.ServeHTTP(&xw, r)
	}
	return http.HandlerFunc(fn)
}

func (s *WebconfigServer) NoAuthMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xw := s.logRequestStarts(w, r)
		defer s.logRequestEnds(&xw, r)
		next.ServeHTTP(&xw, r)
	}
	return http.HandlerFunc(fn)
}

// Token valid and mac must match
func (s *WebconfigServer) CpeMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xw := s.logRequestStarts(w, r)
		defer s.logRequestEnds(&xw, r)

		isValid := false
		token := xw.Token()
		if len(token) > 0 {
			params := mux.Vars(r)
			mac, ok := params["mac"]
			if !ok || len(mac) != 12 {
				Error(&xw, r, http.StatusForbidden, nil)
				return
			}

			if ok, err := s.VerifyCpeToken(token, strings.ToLower(mac)); ok {
				isValid = true
			} else {
				xw.LogDebug(r, "token", fmt.Sprintf("CpeMiddleware() VerifyCpeToken()=false, err=%v", err))
			}
		} else {
			xw.LogDebug(r, "token", "CpeMiddleware() error no token")
		}

		if isValid {
			next.ServeHTTP(&xw, r)
		} else {
			Error(&xw, r, http.StatusForbidden, nil)
		}
	}
	return http.HandlerFunc(fn)
}

// Token valid enough
func (s *WebconfigServer) ApiMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xw := s.logRequestStarts(w, r)
		defer s.logRequestEnds(&xw, r)

		isValid := false
		token := xw.Token()
		if len(token) > 0 {
			if ok, err := s.VerifyApiToken(token); ok {
				isValid = true
			} else {
				xw.LogDebug(r, "token", fmt.Sprintf("ApiMiddleware() VerifyApiToken()=false, err=%v", err))
			}
		} else {
			xw.LogDebug(r, "token", "ApiMiddleware() error no token")
		}

		if isValid {
			next.ServeHTTP(&xw, r)
		} else {
			Error(&xw, r, http.StatusForbidden, nil)
		}
	}
	return http.HandlerFunc(fn)
}

func (s *WebconfigServer) TestingCpeMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xp := NewXpcResponseWriter(w)
		xw := *xp

		// read the token
		authorization := r.Header.Get("Authorization")
		elements := strings.Split(authorization, " ")
		token := ""
		if len(elements) == 2 && elements[0] == "Bearer" {
			token = elements[1]
		}

		isValid := false
		if len(token) > 0 {
			params := mux.Vars(r)
			mac, ok := params["mac"]
			if !ok || len(mac) != 12 {
				Error(&xw, r, http.StatusForbidden, nil)
				return
			}

			if ok, _ := s.VerifyCpeToken(token, strings.ToLower(mac)); ok {
				isValid = true
			}
		}

		if isValid {
			next.ServeHTTP(&xw, r)
		} else {
			Error(&xw, r, http.StatusForbidden, nil)
		}
	}
	return http.HandlerFunc(fn)
}

func (s *WebconfigServer) MetricsEnabled() bool {
	return s.metricsEnabled
}

func (s *WebconfigServer) FactoryResetEnabled() bool {
	return s.factoryResetEnabled
}

func (s *WebconfigServer) SetFactoryResetEnabled(enabled bool) {
	s.factoryResetEnabled = enabled
}

func (s *WebconfigServer) ServerApiTokenAuthEnabled() bool {
	return s.serverApiTokenAuthEnabled
}

func (s *WebconfigServer) SetServerApiTokenAuthEnabled(enabled bool) {
	s.serverApiTokenAuthEnabled = enabled
}

func (s *WebconfigServer) DeviceApiTokenAuthEnabled() bool {
	return s.deviceApiTokenAuthEnabled
}

func (s *WebconfigServer) SetDeviceApiTokenAuthEnabled(enabled bool) {
	s.deviceApiTokenAuthEnabled = enabled
}

func (s *WebconfigServer) TokenApiEnabled() bool {
	return s.tokenApiEnabled
}

func (s *WebconfigServer) SetTokenApiEnabled(enabled bool) {
	s.tokenApiEnabled = enabled
}

func (s *WebconfigServer) TlsConfig() *tls.Config {
	return s.tlsConfig
}

func (s *WebconfigServer) NotLoggedHeaders() []string {
	return s.notLoggedHeaders
}

func (c *WebconfigServer) Poke(cpeMac string, token string, fields log.Fields) (string, error) {
	transactionId, err := c.Patch(cpeMac, token, PokeBody, fields)
	if err != nil {
		return "", common.NewError(err)
	}
	return transactionId, nil
}

func getHeadersForLogAsMap(r *http.Request, notLoggedHeaders []string) map[string]interface{} {
	loggedHeaders := make(map[string]interface{})
	for k, v := range r.Header {
		if util.CaseInsensitiveContains(notLoggedHeaders, k) {
			continue
		}
		loggedHeaders[k] = v
	}
	return loggedHeaders
}

func (s *WebconfigServer) logRequestStarts(w http.ResponseWriter, r *http.Request) XpcResponseWriter {
	remoteIp := r.RemoteAddr
	host := r.Host
	headers := getHeadersForLogAsMap(r, s.notLoggedHeaders)

	// extract the token from the header
	authorization := r.Header.Get("Authorization")
	elements := strings.Split(authorization, " ")
	token := ""
	if len(elements) == 2 && elements[0] == "Bearer" {
		token = elements[1]
	}

	// extract moneytrace from the header
	traceId := ""
	tracePart := strings.Split(r.Header.Get("X-Moneytrace"), ";")[0]
	if elements := strings.Split(tracePart, "="); len(elements) == 2 {
		if elements[0] == "trace-id" {
			traceId = elements[1]
		}
	}

	// extract auditid from the header
	auditId := r.Header.Get("X-Auditid")
	if len(auditId) == 0 {
		auditId = util.GetAuditId()
	}

	fields := log.Fields{
		"path":      r.URL.String(),
		"method":    r.Method,
		"audit_id":  auditId,
		"remote_ip": remoteIp,
		"host_name": host,
		"headers":   headers,
		"logger":    "request",
		"trace_id":  traceId,
	}

	// add cpemac or csid in loggings
	params := mux.Vars(r)
	gtype := params["gtype"]
	switch gtype {
	case "cpe":
		mac := params["gid"]
		mac = strings.ToUpper(mac)
		fields["cpemac"] = mac
	case "configset":
		csid := params["gid"]
		csid = strings.ToLower(csid)
		fields["csid"] = csid
	}
	if mac, ok := params["mac"]; ok {
		mac = strings.ToUpper(mac)
		fields["cpemac"] = mac
	}

	xp := NewXpcResponseWriter(w, time.Now(), token, fields)
	xwriter := *xp

	if r.Method == "POST" {
		var body string
		if r.Body != nil {
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fields["error"] = err
				log.WithFields(fields).Error("request starts")
				return xwriter
			}
			body = string(b)
		}
		xwriter.SetBody(body)
		fields["body"] = body

		contentType := r.Header.Get("Content-type")
		if contentType == "application/msgpack" {
			xwriter.SetBodyObfuscated(true)
		}
	}

	auditFields := xwriter.Audit()
	log.WithFields(auditFields).Info("request starts")

	return xwriter
}

func (s *WebconfigServer) logRequestEnds(xw *XpcResponseWriter, r *http.Request) {
	tdiff := time.Now().Sub(xw.StartTime())
	duration := tdiff.Nanoseconds() / 1000000

	url := r.URL.String()
	response := xw.Response()
	if strings.Contains(url, "/config") || (strings.Contains(url, "/document") && r.Method == "GET") || (url == "/api/v1/token" && r.Method == "POST") {
		response = "****"
	}

	fields := xw.Audit()
	fields["response"] = response
	fields["status"] = xw.Status()
	fields["duration"] = duration
	fields["logger"] = "request"

	log.WithFields(fields).Info("request ends")
}

func LogError(w http.ResponseWriter, r *http.Request, err error) {
	var fields log.Fields
	if xw, ok := w.(*XpcResponseWriter); ok {
		fields = xw.Audit()
		fields["error"] = err
	} else {
		fields = make(log.Fields)
	}

	log.WithFields(fields).Error("internal error")
}

func (xw *XpcResponseWriter) logMessage(r *http.Request, logger string, message string, level int) {
	fields := xw.Audit()
	fields["logger"] = logger

	switch level {
	case LevelWarn:
		log.WithFields(fields).Warn(message)
	case LevelInfo:
		log.WithFields(fields).Info(message)
	case LevelDebug:
		log.WithFields(fields).Debug(message)
	}
}

func (xw *XpcResponseWriter) LogDebug(r *http.Request, logger string, message string) {
	xw.logMessage(r, logger, message, LevelDebug)
}

func (xw *XpcResponseWriter) LogInfo(r *http.Request, logger string, message string) {
	xw.logMessage(r, logger, message, LevelInfo)
}

func (xw *XpcResponseWriter) LogWarn(r *http.Request, logger string, message string) {
	xw.logMessage(r, logger, message, LevelWarn)
}
