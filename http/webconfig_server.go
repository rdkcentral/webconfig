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
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-akka/configuration"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/db/cassandra"
	"github.com/rdkcentral/webconfig/db/sqlite"
	"github.com/rdkcentral/webconfig/security"
	"github.com/rdkcentral/webconfig/tracing"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TODO enum, probably no need
const (
	LevelWarn = iota
	LevelInfo
	LevelDebug
)

const (
	MetricsEnabledDefault                = true
	FactoryResetEnabledDefault           = false
	serverApiTokenAuthEnabledDefault     = false
	deviceApiTokenAuthEnabledDefault     = true
	tokenApiEnabledDefault               = false
	activeDriverDefault                  = "cassandra"
	defaultJwksEnabled                   = false
	defaultTraceparentParentID           = "0000000000000001"
	defaultTracestateVendorID            = "webconfig"
	defaultSupplementaryAppendingEnabled = true
	authPrefixLength                     = 60
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
	wifiSubdocIds = []string{
		"privatessid",
		"homessid",
	}
	codec *security.AesCodec
)

type WebconfigServer struct {
	*http.Server
	db.DatabaseClient
	*security.TokenManager
	*security.JwksManager
	*common.ServerConfig
	*WebpaConnector
	*XconfConnector
	*MqttConnector
	*UpstreamConnector
	sarama.AsyncProducer
	*tracing.XpcTracer
	tlsConfig                     *tls.Config
	notLoggedHeaders              []string
	metricsEnabled                bool
	factoryResetEnabled           bool
	serverApiTokenAuthEnabled     bool
	deviceApiTokenAuthEnabled     bool
	tokenApiEnabled               bool
	kafkaEnabled                  bool
	upstreamEnabled               bool
	appName                       string
	validateMacEnabled            bool
	validPartners                 []string
	jwksEnabled                   bool
	traceparentParentID           string
	tracestateVendorID            string
	supplementaryAppendingEnabled bool
	kafkaProducerEnabled          bool
	kafkaProducerTopic            string
	upstreamProfilesEnabled       bool
	queryParamsValidationEnabled  bool
	minTrust                      int
	validSubdocIdMap              map[string]int
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
	var tokenManager *security.TokenManager

	// setup up database client
	if testOnly {
		dbclient = GetTestDatabaseClient(sc)
	} else {
		dbclient = GetDatabaseClient(sc)
		tokenManager = security.NewTokenManager(conf)
	}

	// setup jwks manager
	jwksEnabled := conf.GetBoolean("webconfig.jwt.api_token.jwks_enabled", defaultJwksEnabled)
	var ctx context.Context
	jwksManager, err := security.NewJwksManager(conf, ctx)
	if jwksEnabled && err != nil {
		if err != nil {
			panic(err)
		}
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

	serverApiTokenAuthEnabled := conf.GetBoolean("webconfig.jwt.server_api_token_auth.enabled", serverApiTokenAuthEnabledDefault)
	deviceApiTokenAuthEnabled := conf.GetBoolean("webconfig.jwt.device_api_token_auth.enabled", deviceApiTokenAuthEnabledDefault)
	tokenApiEnabled := conf.GetBoolean("webconfig.token_api_enabled", tokenApiEnabledDefault)

	var listenHost string
	if conf.GetBoolean("webconfig.server.localhost_only", false) {
		listenHost = "localhost"
	}
	port := conf.GetInt32("webconfig.server.port", 8080)

	kafkaEnabled := conf.GetBoolean("webconfig.kafka.enabled")
	upstreamEnabled := conf.GetBoolean("webconfig.upstream.enabled")
	appName := conf.GetString("webconfig.app_name")
	validateMacEnabled := conf.GetBoolean("webconfig.validate_device_id_as_mac_address", tokenApiEnabledDefault)
	configValidPartners := conf.GetStringList("webconfig.valid_partners")
	validPartners := []string{}
	for _, p := range configValidPartners {
		validPartners = append(validPartners, strings.ToLower(p))
	}

	xpcTracer := tracing.NewXpcTracer(conf)

	supplementaryAppendingEnabled := conf.GetBoolean("webconfig.supplementary_appending_enabled", defaultSupplementaryAppendingEnabled)

	// kafka producer
	var kafkaProducer sarama.AsyncProducer
	kafkaProducerEnabled := conf.GetBoolean("webconfig.kafka_producer.enabled")
	var kafkaProducerTopic string
	if kafkaProducerEnabled {
		brokersStr := conf.GetString("webconfig.kafka_producer.brokers")
		if len(brokersStr) == 0 {
			panic(fmt.Errorf("webconfig.kafka_producer.brokers is empty"))
		}
		brokers := strings.Split(brokersStr, ",")
		kafkaProducerTopic = conf.GetString("webconfig.kafka_producer.topic")

		saramaConfig := sarama.NewConfig()
		saramaConfig.Producer.Return.Errors = true
		kafkaProducer, err = sarama.NewAsyncProducer(brokers, saramaConfig)
		if err != nil {
			panic(err)
		}
	}

	upstreamProfilesEnabled := conf.GetBoolean("webconfig.upstream_profiles_enabled")
	queryParamsValidationEnabled := conf.GetBoolean("webconfig.query_params_validation_enabled")
	minTrust := int(conf.GetInt32("webconfig.min_trust"))
	validSubdocIds := conf.GetStringList("webconfig.valid_subdoc_ids")
	validSubdocIdMap := maps.Clone(common.SubdocBitIndexMap)
	for _, x := range validSubdocIds {
		validSubdocIdMap[x] = 1
	}

	ws := &WebconfigServer{
		Server: &http.Server{
			Addr:         fmt.Sprintf("%v:%v", listenHost, port),
			ReadTimeout:  time.Duration(conf.GetInt32("webconfig.server.read_timeout_in_secs", 3)) * time.Second,
			WriteTimeout: time.Duration(conf.GetInt32("webconfig.server.write_timeout_in_secs", 3)) * time.Second,
		},
		DatabaseClient:                dbclient,
		TokenManager:                  tokenManager,
		JwksManager:                   jwksManager,
		ServerConfig:                  sc,
		WebpaConnector:                NewWebpaConnector(conf, tlsConfig),
		XconfConnector:                NewXconfConnector(conf, tlsConfig),
		MqttConnector:                 NewMqttConnector(conf, tlsConfig),
		UpstreamConnector:             NewUpstreamConnector(conf, tlsConfig),
		AsyncProducer:                 kafkaProducer,
		tlsConfig:                     tlsConfig,
		notLoggedHeaders:              notLoggedHeaders,
		metricsEnabled:                metricsEnabled,
		factoryResetEnabled:           factoryResetEnabled,
		serverApiTokenAuthEnabled:     serverApiTokenAuthEnabled,
		deviceApiTokenAuthEnabled:     deviceApiTokenAuthEnabled,
		tokenApiEnabled:               tokenApiEnabled,
		kafkaEnabled:                  kafkaEnabled,
		upstreamEnabled:               upstreamEnabled,
		appName:                       appName,
		validateMacEnabled:            validateMacEnabled,
		validPartners:                 validPartners,
		jwksEnabled:                   jwksEnabled,
		supplementaryAppendingEnabled: supplementaryAppendingEnabled,
		kafkaProducerEnabled:          kafkaProducerEnabled,
		kafkaProducerTopic:            kafkaProducerTopic,
		upstreamProfilesEnabled:       upstreamProfilesEnabled,
		queryParamsValidationEnabled:  queryParamsValidationEnabled,
		minTrust:                      minTrust,
		validSubdocIdMap:              validSubdocIdMap,
		XpcTracer:                     xpcTracer,
	}

	return ws
}

func (s *WebconfigServer) Stop() {
	s.StopXpcTracer()
}

func (s *WebconfigServer) TestingMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xw := NewXResponseWriter(w)
		metricsAgent := r.Header.Get(common.HeaderMetricsAgent)
		if len(metricsAgent) > 0 {
			xw.SetAuditData("metrics_agent", metricsAgent)
		}

		authorization := r.Header.Get("Authorization")
		if len(authorization) > 0 {
			elements := strings.Split(authorization, " ")
			if len(elements) == 2 && elements[0] == "Bearer" {
				token := elements[1]
				if len(token) > 0 {
					xw.SetToken(token)
				}
			}
		}

		if r.Method == "POST" {
			if r.Body != nil {
				if rbytes, err := io.ReadAll(r.Body); err == nil {
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
		fields := xw.Audit()
		authorization := r.Header.Get("Authorization")
		var tokenErr error
		if len(token) > 0 {
			params := mux.Vars(r)
			mac, ok := params["mac"]
			if !ok || len(mac) != 12 {
				Error(xw, http.StatusForbidden, nil)
				return
			}

			if ok, partnerId, trust, err := s.VerifyCpeToken(token, strings.ToLower(mac)); ok {
				isValid = true
				fields["src_partner"] = partnerId
				fields["trust"] = trust

				if err := s.ValidatePartner(partnerId); err != nil {
					// isValid = false
					partnerId = "unknown"
					tokenErr = common.NewError(err)
				}
				if trust < s.MinTrust() {
					isValid = false
					tokenErr = common.NewError(common.ErrLowTrust)
				}
				xw.SetPartnerId(partnerId)
			} else {
				tokenErr = common.NewError(err)
			}
		} else {
			tokenErr = common.NewError(errors.New("CpeMiddleware() error no token"))
		}

		if tokenErr != nil {
			fields["error"] = tokenErr
		}

		if isValid {
			next.ServeHTTP(xw, r)
		} else {
			s.LogToken(xw, authorization, token, tokenErr)
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
			var kid string
			if x, err := security.ParseKidFromTokenHeader(token); err == nil {
				kid = x
			}
			fields := xw.Audit()
			tfields := common.FilterLogFields(fields)
			tfields["logger"] = "token"
			tfields["kid"] = kid

			if ok, err := s.VerifyApiToken(token); ok {
				isValid = true
				log.WithFields(tfields).Debug("valid")
			} else {
				tfields["error"] = fmt.Sprintf("ApiMiddleware::VerifyApiToken() %v", err)
				log.WithFields(tfields).Debug("rejected")
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
		xw := NewXResponseWriter(w)

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

			if ok, _, _, _ := s.VerifyCpeToken(token, strings.ToLower(mac)); ok {
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

func (s *WebconfigServer) VerifyApiToken(tokenStr string) (bool, error) {
	if s.JwksEnabled() {
		if _, err := s.JwksManager.VerifyApiToken(tokenStr); err != nil {
			return false, common.NewError(err)
		}
	} else {
		if _, err := s.TokenManager.VerifyApiToken(tokenStr); err != nil {
			return false, common.NewError(err)
		}
	}
	return true, nil
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

func (s *WebconfigServer) AppName() string {
	return s.appName
}

func (s *WebconfigServer) SetAppName(appName string) {
	s.appName = appName
}

func (s *WebconfigServer) ValidateMacEnabled() bool {
	return s.validateMacEnabled
}

func (s *WebconfigServer) SetValidateMacEnabled(validateMacEnabled bool) {
	s.validateMacEnabled = validateMacEnabled
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

func (s *WebconfigServer) ValidPartners() []string {
	return s.validPartners
}

func (s *WebconfigServer) SetValidPartners(validPartners []string) {
	s.validPartners = validPartners
}

func (s *WebconfigServer) JwksEnabled() bool {
	return s.jwksEnabled
}

func (s *WebconfigServer) SetJwksEnabled(enabled bool) {
	s.jwksEnabled = enabled
}

func (s *WebconfigServer) TraceparentParentID() string {
	return s.traceparentParentID
}

func (s *WebconfigServer) SetTraceparentParentID(x string) {
	s.traceparentParentID = x
}

func (s *WebconfigServer) TracestateVendorID() string {
	return s.tracestateVendorID
}

func (s *WebconfigServer) SetTracestateVendorID(x string) {
	s.tracestateVendorID = x
}

func (s *WebconfigServer) SupplementaryAppendingEnabled() bool {
	return s.supplementaryAppendingEnabled
}

func (s *WebconfigServer) SetSupplementaryAppendingEnabled(enabled bool) {
	s.supplementaryAppendingEnabled = enabled
}

func (s *WebconfigServer) KafkaProducerEnabled() bool {
	return s.kafkaProducerEnabled
}

func (s *WebconfigServer) SetKafkaProducerEnabled(enabled bool) {
	s.kafkaProducerEnabled = enabled
}

func (s *WebconfigServer) KafkaProducerTopic() string {
	return s.kafkaProducerTopic
}

func (s *WebconfigServer) SetKafkaProducerTopic(x string) {
	s.kafkaProducerTopic = x
}

func (s *WebconfigServer) UpstreamProfilesEnabled() bool {
	return s.upstreamProfilesEnabled
}

func (s *WebconfigServer) SetUpstreamProfilesEnabled(enabled bool) {
	s.upstreamProfilesEnabled = enabled
}

func (s *WebconfigServer) QueryParamsValidationEnabled() bool {
	return s.queryParamsValidationEnabled
}

func (s *WebconfigServer) SetQueryParamsValidationEnabled(enabled bool) {
	s.queryParamsValidationEnabled = enabled
}

func (s *WebconfigServer) MinTrust() int {
	return s.minTrust
}

func (s *WebconfigServer) SetMinTrust(trust int) {
	s.minTrust = trust
}

func (s *WebconfigServer) ValidSubdocIdMap() map[string]int {
	return s.validSubdocIdMap
}

func (s *WebconfigServer) SetValidSubdocIdMap(x map[string]int) {
	s.validSubdocIdMap = x
}

func (s *WebconfigServer) ValidatePartner(parsedPartner string) error {
	// if no valid partners are configured, all partners are accepted/validated
	if len(s.validPartners) == 0 {
		return nil
	}

	partner := strings.ToLower(parsedPartner)
	for _, p := range s.validPartners {
		if partner == p {
			return nil
		}
	}
	return fmt.Errorf("invalid partner")
}

func (c *WebconfigServer) Poke(rHeader http.Header, cpeMac string, token string, pokeStr string, fields log.Fields) (string, error) {
	body := fmt.Sprintf(common.PokeBodyTemplate, pokeStr)
	transactionId, err := c.Patch(rHeader, cpeMac, token, []byte(body), fields)
	if err != nil {
		return "", common.NewError(err)
	}
	return transactionId, nil
}

func getFilteredHeader(r *http.Request, notLoggedHeaders []string) http.Header {
	header := r.Header.Clone()
	for _, k := range notLoggedHeaders {
		header.Del(k)
	}
	return header
}

func (s *WebconfigServer) logRequestStarts(w http.ResponseWriter, r *http.Request) *XResponseWriter {
	remoteIp := r.RemoteAddr
	host := r.Host
	header := getFilteredHeader(r, s.notLoggedHeaders)

	// extract the token from the header
	authorization := r.Header.Get("Authorization")
	elements := strings.Split(authorization, " ")
	token := ""
	if len(elements) == 2 && elements[0] == "Bearer" {
		token = elements[1]
	}

	var xmTraceId string

	// extract moneytrace from the header
	tracePart := strings.Split(r.Header.Get("X-Moneytrace"), ";")[0]
	if elements := strings.Split(tracePart, "="); len(elements) == 2 {
		if elements[0] == "trace-id" {
			xmTraceId = elements[1]
		}
	}

	// extract auditid from the header
	auditId := r.Header.Get("X-Auditid")
	if len(auditId) == 0 {
		auditId = util.GetAuditId()
	}

	// traceparent handling for E2E tracing
	xpcTrace := tracing.NewXpcTrace(s.XpcTracer, r)
	traceId := xpcTrace.TraceID
	if len(traceId) == 0 {
		traceId = xmTraceId
	}

	headerMap := util.HeaderToMap(header)
	fields := log.Fields{
		"path":              r.URL.String(),
		"method":            r.Method,
		"audit_id":          auditId,
		"remote_ip":         remoteIp,
		"host_name":         host,
		"header":            headerMap,
		"logger":            "request",
		"trace_id":          traceId,
		"app_name":          s.AppName(),
		"traceparent":       xpcTrace.ReqTraceparent,
		"tracestate":        xpcTrace.ReqTracestate,
		"out_traceparent":   xpcTrace.OutTraceparent,
		"out_tracestate":    xpcTrace.OutTracestate,
		"req_moracide_tags": xpcTrace.ReqMoracideTags,
		"xpc_trace":         xpcTrace,
	}

	userAgent := r.UserAgent()
	if len(userAgent) > 0 {
		fields["user_agent"] = userAgent
	}
	if x := r.Header.Get(common.HeaderMetricsAgent); len(x) > 0 {
		fields["metrics_agent"] = x
	}
	if x := r.Header.Get("X-Webconfig-Transaction-Id"); len(x) > 0 {
		fields["webconfig_transaction_id"] = x
	}
	if x := r.Header.Get(common.HeaderSourceAppName); len(x) > 0 {
		fields["src_app_name"] = x
	}
	if len(xmTraceId) > 0 {
		fields["xmoney_trace_id"] = xmTraceId
	}

	// add cpemac or csid in loggings
	params := mux.Vars(r)
	gtype := params["gtype"]
	switch gtype {
	case "cpe":
		mac := params["gid"]
		mac = strings.ToUpper(mac)
		fields["cpemac"] = mac
		fields["cpe_mac"] = mac
	case "configset":
		csid := params["gid"]
		csid = strings.ToLower(csid)
		fields["csid"] = csid
	}
	if mac, ok := params["mac"]; ok {
		mac = strings.ToUpper(mac)
		fields["cpemac"] = mac
		fields["cpe_mac"] = mac
	}

	xwriter := NewXResponseWriter(w, time.Now(), token, fields)

	if r.Method == "POST" {
		if r.Body != nil {
			bbytes, err := io.ReadAll(r.Body)
			if err != nil {
				fields["error"] = err
				log.WithFields(fields).Error("request starts")
				return xwriter
			}
			xwriter.SetBodyBytes(bbytes)
		}
	}

	if userAgent != "mget" {
		tfields := common.FilterLogFields(fields)
		log.WithFields(tfields).Info("request starts")
	}

	xwriter.LogDebug(r, "tracing", fmt.Sprintf("Trace final out_traceparent %s out_traceState %s", xpcTrace.OutTraceparent, xpcTrace.OutTracestate))
	return xwriter
}

func (s *WebconfigServer) logRequestEnds(xw *XResponseWriter, r *http.Request) {
	tdiff := time.Now().Sub(xw.StartTime())
	duration := tdiff.Nanoseconds() / 1000000

	url := r.URL.String()
	fields := xw.Audit()
	if strings.Contains(url, "/config") {
		rbytes := []byte(xw.Response())
		var isTelemetry bool
		if itf, ok := fields["is_telemetry"]; ok {
			isTelemetry = itf.(bool)
		}

		if !isTelemetry && xw.Status() == http.StatusOK {
			resHeader := xw.ResponseWriter.Header()
			if mpdict, err := util.ParseMultipartsForLogging(rbytes, resHeader, wifiSubdocIds); err == nil {
				fields["response"] = mpdict
			}
		} else {
			res_itf, res_text := GetResponseLogObjs(rbytes)
			fields["response"] = res_itf
			fields["response_text"] = res_text
		}

		var doc_map util.Dict
		if itf, ok := fields["document"]; ok {
			if d, ok := itf.(util.Dict); ok {
				doc_map = d
			}
		}
		if len(doc_map) > 0 {
			tfields := common.FilterLogFields(fields)
			tfields["logger"] = "doc"
			log.WithFields(tfields).Debug("details")
		}
		if xw.Status() < 500 {
			delete(fields, "document")
		}
	} else if (strings.Contains(url, "/document") && r.Method == "GET") || (url == "/api/v1/token" && r.Method == "POST") {
		fields["response"] = ObfuscatedMap
		fields["response_text"] = "****"
	} else {
		_, ok := fields["response"]
		if !ok {
			response := xw.Response()
			if len(response) > 0 {
				var itf interface{}
				err := json.Unmarshal([]byte(response), &itf)
				if err != nil {
					err1 := common.NewError(err)
					fields["response"] = ObfuscatedMap
					fields["response_text"] = err1.Error()
				} else {
					fields["response"] = itf
				}
			} else {
				fields["response"] = ObfuscatedMap
				fields["response_text"] = ""
			}
		}
	}

	fields["status"] = xw.Status()
	fields["duration"] = duration
	fields["logger"] = "request"

	s.XpcTracer.SetSpan(fields, s.XpcTracer.MoracideTagPrefix())

	var userAgent string
	if itf, ok := fields["user_agent"]; ok {
		userAgent = itf.(string)
	}
	if userAgent != "mget" {
		tfields := common.FilterLogFields(fields)
		log.WithFields(tfields).Info("request ends")
	}
}

func LogError(w http.ResponseWriter, err error) {
	var fields log.Fields
	if xw, ok := w.(*XResponseWriter); ok {
		fields = xw.Audit()
		fields["error"] = err
	} else {
		fields = make(log.Fields)
	}

	log.WithFields(fields).Error("internal error")
}

func (xw *XResponseWriter) logMessage(logger string, message string, level int) {
	fields := xw.Audit()
	tfields := common.FilterLogFields(fields)
	tfields["logger"] = logger

	switch level {
	case LevelWarn:
		log.WithFields(tfields).Warn(message)
	case LevelInfo:
		log.WithFields(tfields).Info(message)
	case LevelDebug:
		log.WithFields(tfields).Debug(message)
	}
}

// REMINDER use by the middleware functions
func (xw *XResponseWriter) LogDebug(r *http.Request, logger string, message string) {
	xw.logMessage(logger, message, LevelDebug)
}

func (xw *XResponseWriter) LogInfo(r *http.Request, logger string, message string) {
	xw.logMessage(logger, message, LevelInfo)
}

func (xw *XResponseWriter) LogWarn(r *http.Request, logger string, message string) {
	xw.logMessage(logger, message, LevelWarn)
}

func GetResponseLogObjs(rbytes []byte) (interface{}, string) {
	if len(rbytes) == 0 {
		return EmptyMap, ""
	}

	if !util.IsValidUTF8(rbytes) {
		return ObfuscatedMap, "****"
	}

	var itf interface{}
	err := json.Unmarshal(rbytes, &itf)
	if err != nil {
		return BadJsonResponseMap, string(rbytes)
	}
	return itf, ""
}

func (s *WebconfigServer) ForwardKafkaMessage(kbytes []byte, m *common.EventMessage, fields log.Fields) {
	tfields := common.CopyCoreLogFields(fields)

	bbytes, err := json.Marshal(m)
	if err != nil {
		tfields["logger"] = "error"
		log.WithFields(tfields).Error(common.NewError(err))
		return
	}
	outMessage := &sarama.ProducerMessage{
		Topic: s.KafkaProducerTopic(),
		Key:   sarama.ByteEncoder(kbytes),
		Value: sarama.ByteEncoder(bbytes),
	}
	s.Input() <- outMessage

	tfields["logger"] = "kafkaproducer"
	tfields["output_topic"] = outMessage.Topic
	tfields["output_key"] = string(kbytes)
	tfields["output_body"] = m
	log.WithFields(tfields).Info("send")
}

func (s *WebconfigServer) ForwardSuccessKafkaMessages(messages []common.EventMessage, fields log.Fields) {
	tfields := common.CopyCoreLogFields(fields)
	tfields["logger"] = "kafkaproducer"
	tfields["output_topic"] = s.KafkaProducerTopic()

	for _, m := range messages {
		if len(m.DeviceId) != 16 {
			log.WithFields(tfields).Warn("invalid device_id " + m.DeviceId)
			continue
		}
		mac := m.DeviceId[4:]
		transactionUuid := s.AppName() + "_____" + uuid.New().String()
		m.TransactionUuid = &transactionUuid

		bbytes, err := json.Marshal(m)
		if err != nil {
			tfields["logger"] = "error"
			log.WithFields(tfields).Error(common.NewError(err))
			return
		}
		outMessage := &sarama.ProducerMessage{
			Topic: s.KafkaProducerTopic(),
			Key:   sarama.ByteEncoder(strings.ToLower(mac)),
			Value: sarama.ByteEncoder(bbytes),
		}
		s.Input() <- outMessage

		tfields["output_key"] = mac
		tfields["output_body"] = m
		log.WithFields(tfields).Info("send")
	}
}

func (s *WebconfigServer) LogToken(xw *XResponseWriter, authorization, token string, tokenErr error) {
	fields := xw.Audit()
	fields["logger"] = "token"
	tfields := common.FilterLogFields(fields)
	var headerMap map[string]string
	var isObfuscated bool
	if itf, ok := tfields["header"]; ok {
		headerMap = itf.(map[string]string)
		if len(headerMap) > 0 {
			ss := authorization
			if len(ss) > authPrefixLength {
				ss = authorization[:authPrefixLength] + "****"
				isObfuscated = true
			}
			headerMap["Authorization"] = ss
		}
	}

	if isObfuscated {
		if codec == nil {
			codec, _ = security.NewAesCodec(s.Config)
		}

		if codec == nil {
			tfields["plaintoken"] = token
		} else {
			var encToken string
			if encryptedB64, err := codec.Encrypt(token); err == nil {
				encToken = encryptedB64
			}
			tfields["enctoken"] = encToken
		}
	}

	log.WithFields(tfields).Debug(tokenErr)
}

func (s *WebconfigServer) HandleKafkaProducerResults() {
	if s.AsyncProducer == nil {
		return
	}

	for {
		select {
		case success := <-s.Successes():
			if success == nil {
				continue
			}
			fields := make(log.Fields)
			fields["logger"] = "kafkaproducer"
			fields["output_topic"] = success.Topic
			fields["output_partition"] = success.Partition
			fields["output_offset"] = success.Offset
			log.WithFields(fields).Debug("sent")
		case pErr := <-s.Errors():
			if pErr == nil || pErr.Msg == nil {
				continue
			}
			if m := s.Metrics(); m != nil {
				m.ObserveKafkaProducerErr(pErr.Msg.Topic, pErr.Msg.Partition)
			}
			fields := make(log.Fields)
			fields["logger"] = "kafkaproducer"
			fields["output_topic"] = pErr.Msg.Topic
			fields["output_partition"] = pErr.Msg.Partition
			kbytes, err := pErr.Msg.Key.Encode()
			if err != nil {
				log.WithFields(fields).Error(common.NewError(err))
			} else {
				fields["output_key"] = string(kbytes)
			}

			vbytes, err := pErr.Msg.Value.Encode()
			if err != nil {
				log.WithFields(fields).Error(common.NewError(err))
			} else {
				var itf interface{}
				err1 := json.Unmarshal(vbytes, &itf)
				if err1 != nil {
					log.WithFields(fields).Error(common.NewError(err1))
					fields["output_body_text"] = base64.StdEncoding.EncodeToString(vbytes)
				} else {
					fields["output_body"] = itf
				}
			}

			log.WithFields(fields).Error(pErr.Err)
		}
	}
}

func (s *WebconfigServer) StopXpcTracer() {
	sdkTraceProvider, ok := s.XpcTracer.OtelTracerProvider().(*sdktrace.TracerProvider)
	if ok && sdkTraceProvider != nil {
		sdkTraceProvider.Shutdown(context.TODO())
	}
}

func (s *WebconfigServer) SpanMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.XpcTracer.OtelEnabled {
			ctx, otelSpan := tracing.NewOtelSpan(s.XpcTracer, r)
			r = r.WithContext(ctx)
			defer tracing.EndOtelSpan(s.XpcTracer, otelSpan)
		}
		next.ServeHTTP(w, r)
	})
}
