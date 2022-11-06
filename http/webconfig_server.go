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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-akka/configuration"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/db/cassandra"
	"github.com/rdkcentral/webconfig/db/sqlite"
	"github.com/rdkcentral/webconfig/security"
	"github.com/rdkcentral/webconfig/util"
)

// TODO enum, probably no need
const (
	LevelWarn = iota
	LevelInfo
	LevelDebug
)

const (
	MetricsEnabledDefault            = true
	FactoryResetEnabledDefault       = false
	serverApiTokenAuthEnabledDefault = false
	deviceApiTokenAuthEnabledDefault = true
	tokenApiEnabledDefault           = false
	activeDriverDefault              = "cassandra"
)

var (
	selectedHeaders = []string{
		"If-None-Match",
		"X-System-Firmware-Version",
		"X-System-Supported-Docs",
		"X-System-Supplementaryservice-Sync",
		"X-System-Model-Name",
		"X-System-Product-Class",
		"X-System-Schema-Version",
	}
)

type WebconfigServer struct {
	*http.Server
	db.DatabaseClient
	*security.TokenManager
	*common.ServerConfig
	*WebpaConnector
	*CodebigConnector
	*XconfConnector
	*MqttConnector
	*UpstreamConnector
	tlsConfig                 *tls.Config
	notLoggedHeaders          []string
	metricsEnabled            bool
	factoryResetEnabled       bool
	serverApiTokenAuthEnabled bool
	deviceApiTokenAuthEnabled bool
	tokenApiEnabled           bool
	blockedSubdocIds          []string
	kafkaEnabled              bool
	upstreamEnabled           bool
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

func GetTestDatabaseClient(sc *common.ServerConfig) db.DatabaseClient {
	// TODO check the client init for enabled
	var tdbclient db.DatabaseClient
	var err error

	// this is meant to override the database.active_driver
	activeDriver := sc.GetString("webconfig.database.active_driver", activeDriverDefault)
	if x := os.Getenv("TESTDB_DRIVER"); len(x) > 0 {
		activeDriver = x
	}

	switch activeDriver {
	case "sqlite":
		tdbclient, err = sqlite.GetTestSqliteClient(sc.Config, true)
		if err != nil {
			panic(err)
		}
	case "cassandra", "yugabyte":
		tdbclient, err = cassandra.GetTestCassandraClient(sc.Config, true)
		if err != nil {
			panic(err)
		}
	default:
		err = fmt.Errorf("Unsupported database.active_driver %v is configured", activeDriver)
		panic(err)
	}
	err = tdbclient.SetUp()
	if err != nil {
		panic(err)
	}
	return tdbclient
}

func GetDatabaseClient(sc *common.ServerConfig) db.DatabaseClient {
	var dbclient db.DatabaseClient
	var err error

	activeDriver := sc.GetString("webconfig.database.active_driver", activeDriverDefault)
	switch activeDriver {
	case "sqlite":
		dbclient, err = sqlite.NewSqliteClient(sc.Config, false)
		if err != nil {
			panic(err)
		}
	case "cassandra", "yugabyte":
		dbclient, err = cassandra.NewCassandraClient(sc.Config, false)
		if err != nil {
			panic(err)
		}
	default:
		err = fmt.Errorf("Unsupported database.active_driver %v is configured", activeDriver)
		panic(err)
	}

	// WARNING unlike the testclient, dbclient (used by the application)
	// chooses NOT to run SetUp(). It leaves devops/dba to prepare the db

	return dbclient
}

// testOnly=true ==> running unit test
func NewWebconfigServer(sc *common.ServerConfig, testOnly bool) *WebconfigServer {
	conf := sc.Config
	var dbclient db.DatabaseClient

	// setup up database client
	if testOnly {
		dbclient = GetTestDatabaseClient(sc)
	} else {
		dbclient = GetDatabaseClient(sc)
	}

	metricsEnabled := conf.GetBoolean("webconfig.server.metrics_enabled", MetricsEnabledDefault)
	factoryResetEnabled := conf.GetBoolean("webconfig.factory_reset_enabled", FactoryResetEnabledDefault)

	// configure headers that should not be logged
	ignoredHeaders := conf.GetStringList("webconfig.log.ignored_headers")
	ignoredHeaders = append(common.DefaultIgnoredHeaders, ignoredHeaders...)
	var notLoggedHeaders []string
	for _, x := range ignoredHeaders {
		notLoggedHeaders = append(notLoggedHeaders, strings.ToLower(x))
	}

	// tlsConfig, here we ignore any error
	tlsConfig, _ := NewTlsConfig(conf)

	panicExitEnabled := conf.GetBoolean("webconfig.panic_exit_enabled", false)
	// load codebig credentials
	satClientId := os.Getenv("SAT_CLIENT_ID")
	if len(satClientId) == 0 {
		if panicExitEnabled {
			panic("No env SAT_CLIENT_ID")
		}
	}

	satClientSecret := os.Getenv("SAT_CLIENT_SECRET")
	if len(satClientSecret) == 0 {
		if panicExitEnabled {
			panic("No env SAT_CLIENT_SECRET")
		}
	}

	serverApiTokenAuthEnabled := conf.GetBoolean("webconfig.jwt.server_api_token_auth.enabled", serverApiTokenAuthEnabledDefault)
	deviceApiTokenAuthEnabled := conf.GetBoolean("webconfig.jwt.device_api_token_auth.enabled", deviceApiTokenAuthEnabledDefault)
	tokenApiEnabled := conf.GetBoolean("webconfig.token_api_enabled", tokenApiEnabledDefault)
	blockedSubdocIds := conf.GetStringList("webconfig.blocked_subdoc_ids")

	var listenHost string
	if conf.GetBoolean("webconfig.server.localhost_only", false) {
		listenHost = "localhost"
	}
	port := conf.GetInt32("webconfig.server.port", 8080)

	kafkaEnabled := conf.GetBoolean("webconfig.kafka.enabled")
	upstreamEnabled := conf.GetBoolean("webconfig.upstream.enabled")

	return &WebconfigServer{
		Server: &http.Server{
			Addr:         fmt.Sprintf("%v:%v", listenHost, port),
			ReadTimeout:  time.Duration(conf.GetInt32("webconfig.server.read_timeout_in_secs", 3)) * time.Second,
			WriteTimeout: time.Duration(conf.GetInt32("webconfig.server.write_timeout_in_secs", 3)) * time.Second,
		},
		DatabaseClient:            dbclient,
		TokenManager:              security.NewTokenManager(conf),
		ServerConfig:              sc,
		WebpaConnector:            NewWebpaConnector(conf, tlsConfig),
		CodebigConnector:          NewCodebigConnector(conf, satClientId, satClientSecret, tlsConfig),
		XconfConnector:            NewXconfConnector(conf, tlsConfig),
		MqttConnector:             NewMqttConnector(conf, tlsConfig),
		UpstreamConnector:         NewUpstreamConnector(conf, tlsConfig),
		tlsConfig:                 tlsConfig,
		notLoggedHeaders:          notLoggedHeaders,
		metricsEnabled:            metricsEnabled,
		factoryResetEnabled:       factoryResetEnabled,
		serverApiTokenAuthEnabled: serverApiTokenAuthEnabled,
		deviceApiTokenAuthEnabled: deviceApiTokenAuthEnabled,
		tokenApiEnabled:           tokenApiEnabled,
		blockedSubdocIds:          blockedSubdocIds,
		kafkaEnabled:              kafkaEnabled,
		upstreamEnabled:           upstreamEnabled,
	}
}

func (s *WebconfigServer) TestingMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xw := NewXpcResponseWriter(w)
		metricsAgent := r.Header.Get(common.HeaderMetricsAgent)
		if len(metricsAgent) > 0 {
			xw.SetAuditData("metrics_agent", metricsAgent)
		}

		if r.Method == "POST" {
			if r.Body != nil {
				if rbytes, err := ioutil.ReadAll(r.Body); err == nil {
					xw.SetBodyBytes(rbytes)
				}
			}
		}
		next.ServeHTTP(xw, r)
	}
	return http.HandlerFunc(fn)
}

func (s *WebconfigServer) NoAuthMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xw := s.logRequestStarts(w, r)
		defer s.logRequestEnds(xw, r)
		next.ServeHTTP(xw, r)
	}
	return http.HandlerFunc(fn)
}

// Token valid and mac must match
func (s *WebconfigServer) CpeMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xw := s.logRequestStarts(w, r)
		defer s.logRequestEnds(xw, r)

		isValid := false
		token := xw.Token()
		if len(token) > 0 {
			params := mux.Vars(r)
			mac, ok := params["mac"]
			if !ok || len(mac) != 12 {
				Error(xw, http.StatusForbidden, nil)
				return
			}

			if ok, partnerId, err := s.VerifyCpeToken(token, strings.ToLower(mac)); ok {
				isValid = true
				xw.SetPartnerId(partnerId)
			} else {
				xw.LogDebug(r, "token", fmt.Sprintf("CpeMiddleware() VerifyCpeToken()=false, err=%v", err))
			}
		} else {
			xw.LogDebug(r, "token", "CpeMiddleware() error no token")
		}

		if isValid {
			next.ServeHTTP(xw, r)
		} else {
			Error(xw, http.StatusForbidden, nil)
		}
	}
	return http.HandlerFunc(fn)
}

// Token valid enough
func (s *WebconfigServer) ApiMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xw := s.logRequestStarts(w, r)
		defer s.logRequestEnds(xw, r)

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
			next.ServeHTTP(xw, r)
		} else {
			Error(xw, http.StatusForbidden, nil)
		}
	}
	return http.HandlerFunc(fn)
}

func (s *WebconfigServer) TestingCpeMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xw := NewXpcResponseWriter(w)

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
				Error(xw, http.StatusForbidden, nil)
				return
			}

			if ok, _, _ := s.VerifyCpeToken(token, strings.ToLower(mac)); ok {
				isValid = true
			}
		}

		if isValid {
			next.ServeHTTP(xw, r)
		} else {
			Error(xw, http.StatusForbidden, nil)
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

func (s *WebconfigServer) BlockedSubdocIds() []string {
	return s.blockedSubdocIds
}

func (s *WebconfigServer) SetBlockedSubdocIds(blockedSubdocIds []string) {
	s.blockedSubdocIds = blockedSubdocIds
}

func (s *WebconfigServer) KafkaEnabled() bool {
	return s.kafkaEnabled
}

func (s *WebconfigServer) SetKafkaEnabled(enabled bool) {
	s.kafkaEnabled = enabled
}

func (s *WebconfigServer) UpstreamEnabled() bool {
	return s.upstreamEnabled
}

func (s *WebconfigServer) SetUpstreamEnabled(enabled bool) {
	s.upstreamEnabled = enabled
}

func (s *WebconfigServer) GetUpstreamConnector() *UpstreamConnector {
	if !s.upstreamEnabled {
		return nil
	}
	return s.UpstreamConnector
}

func (s *WebconfigServer) TlsConfig() *tls.Config {
	return s.tlsConfig
}

func (s *WebconfigServer) NotLoggedHeaders() []string {
	return s.notLoggedHeaders
}

func (s *WebconfigServer) NewMetrics() *common.AppMetrics {
	m := common.NewMetrics()
	sclient, ok := s.DatabaseClient.(*sqlite.SqliteClient)
	if ok {
		sclient.SetMetrics(m)
	}
	return m
}

func (c *WebconfigServer) Poke(cpeMac string, token string, pokeStr string, fields log.Fields) (string, error) {
	body := fmt.Sprintf(common.PokeBodyTemplate, pokeStr)
	transactionId, err := c.Patch(cpeMac, token, []byte(body), fields)
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

func (s *WebconfigServer) logRequestStarts(w http.ResponseWriter, r *http.Request) *XpcResponseWriter {
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
		"app_name":  "webconfig",
	}

	userAgent := r.UserAgent()
	if len(userAgent) > 0 {
		fields["user_agent"] = userAgent
	}
	metricsAgent := r.Header.Get(common.HeaderMetricsAgent)
	if len(metricsAgent) > 0 {
		fields["metrics_agent"] = metricsAgent
	}

	// log critical headers
	_, ok := headers["If-None-Match"]
	if ok {
		selected := util.Dict{}
		for _, k := range selectedHeaders {
			selected[k] = headers[k]
		}
		fields["selected"] = selected
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

	xwriter := NewXpcResponseWriter(w, time.Now(), token, fields)

	if r.Method == "POST" {
		if r.Body != nil {
			bbytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fields["error"] = err
				log.WithFields(fields).Error("request starts")
				return xwriter
			}
			xwriter.SetBodyBytes(bbytes)
		}
	}

	auditFields := xwriter.Audit()

	if userAgent != "mget" {
		log.WithFields(auditFields).Info("request starts")
	}

	return xwriter
}

func (s *WebconfigServer) logRequestEnds(xw *XpcResponseWriter, r *http.Request) {
	tdiff := time.Now().Sub(xw.StartTime())
	duration := tdiff.Nanoseconds() / 1000000

	url := r.URL.String()
	fields := xw.Audit()
	if strings.Contains(url, "/config") || (strings.Contains(url, "/document") && r.Method == "GET") || (url == "/api/v1/token" && r.Method == "POST") {
		fields["response"] = ObfuscatedMap
		fields["response_text"] = "****"
	} else {
		_, ok := fields["response"]
		// XPC-13444 if the "response" is already set in the audit, then no need to do more handling
		if !ok {
			response := xw.Response()
			var itf interface{}
			err := json.Unmarshal([]byte(response), &itf)
			if err != nil {
				err1 := common.NewError(err)
				fields["response"] = ObfuscatedMap
				fields["response_text"] = err1.Error()
			}
		}
	}

	fields["status"] = xw.Status()
	fields["duration"] = duration
	fields["logger"] = "request"

	var userAgent string
	if itf, ok := fields["user_agent"]; ok {
		userAgent = itf.(string)
	}
	if userAgent != "mget" {
		log.WithFields(fields).Info("request ends")
	}
}

func LogError(w http.ResponseWriter, err error) {
	var fields log.Fields
	if xw, ok := w.(*XpcResponseWriter); ok {
		fields = xw.Audit()
		fields["error"] = err
	} else {
		fields = make(log.Fields)
	}

	log.WithFields(fields).Error("internal error")
}

func (xw *XpcResponseWriter) logMessage(logger string, message string, level int) {
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

// REMINDER use by the middleware functions
func (xw *XpcResponseWriter) LogDebug(r *http.Request, logger string, message string) {
	xw.logMessage(logger, message, LevelDebug)
}

func (xw *XpcResponseWriter) LogInfo(r *http.Request, logger string, message string) {
	xw.logMessage(logger, message, LevelInfo)
}

func (xw *XpcResponseWriter) LogWarn(r *http.Request, logger string, message string) {
	xw.logMessage(logger, message, LevelWarn)
}
