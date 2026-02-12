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
package cassandra

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-akka/configuration"
	"github.com/gocql/gocql"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/security"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
)

const (
	ProtocolVersion               = 4
	DefaultKeyspace               = "webconfig"
	DefaultTestKeyspace           = "test_webconfig"
	DisableInitialHostLookup      = false
	DefaultSleepTimeInMillisecond = 10
	DefaultConnections            = 2
	DefaultPageSize               = 50
)

// if 'wifi_schema_v2_enabled'=true, v1.3 is also supported
type CassandraClient struct {
	db.BaseClient
	*gocql.Session
	*gocql.ClusterConfig
	*security.AesCodec
	*common.AppMetrics
	concurrentQueries                chan bool
	localDc                          string
	blockedSubdocIds                 []string
	encryptedSubdocIds               []string
	stateCorrectionEnabled           bool
	lockRootDocumentEnabled          bool
	supplementaryPrecookEnabled      bool
	supplementaryPrecookStateTTLDays int
}

/*
current column types:
      2 columnType=bigint
     44 columnType=boolean
     37 columnType=int
      2 columnType=list
     90 columnType=text
      5 columnType=timestamp
     13 columnType=uuid
*/

func NewCassandraClient(conf *configuration.Config, testOnly bool) (*CassandraClient, error) {
	var codec *security.AesCodec
	var err error

	dbdriver := "cassandra"
	if x := conf.GetString("webconfig.database.active_driver"); x == "yugabyte" {
		dbdriver = "yugabyte"
	}

	// build codec
	if testOnly {
		codec = security.NewTestCodec(conf)
		if x := os.Getenv("TESTDB_DRIVER"); x == "yugabyte" {
			dbdriver = x
		}
	} else {
		codec, err = security.NewAesCodec(conf)
		if err != nil {
			return nil, common.NewError(err)
		}
	}

	dbconf := conf.GetConfig("webconfig.database." + dbdriver)

	// init
	hosts := dbconf.GetStringList("hosts")
	cluster := gocql.NewCluster(hosts...)

	cluster.Consistency = gocql.LocalQuorum
	cluster.ProtoVersion = ProtocolVersion
	cluster.DisableInitialHostLookup = DisableInitialHostLookup
	cluster.Timeout = time.Duration(dbconf.GetInt32("timeout_in_sec", 1)) * time.Second
	cluster.ConnectTimeout = time.Duration(dbconf.GetInt32("connect_timeout_in_sec", 1)) * time.Second
	cluster.NumConns = int(dbconf.GetInt32("connections", DefaultConnections))

	cluster.RetryPolicy = &gocql.DowngradingConsistencyRetryPolicy{
		ConsistencyLevelsToTry: []gocql.Consistency{
			gocql.LocalQuorum,
			gocql.LocalOne,
			gocql.One,
		},
	}

	localDc := dbconf.GetString("local_dc")
	if len(localDc) > 0 {
		cluster.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(localDc)
	}

	var password string
	encryptedPassword := os.Getenv("ENCRYPTED_PASSWORD")
	if len(encryptedPassword) == 0 {
		encryptedPassword = dbconf.GetString("encrypted_password")
	}
	user := dbconf.GetString("user")
	isSslEnabled := dbconf.GetBoolean("is_ssl_enabled")

	// if the password is encrypted, we need to decrypt it
	if encryptedPassword != "" {
		password, err = codec.Decrypt(encryptedPassword)
		if err != nil {
			return nil, common.NewError(err)
		}
	} else {
		password = dbconf.GetString("password")
	}

	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: user,
		Password: password,
	}

	if isSslEnabled {
		tlsConfig, err := loadCassandraTLSConfig(dbconf, dbdriver)
		if err != nil {
			return nil, common.NewError(err)
		}

		sslOpts := &gocql.SslOptions{
			Config:                 tlsConfig,
			EnableHostVerification: false,
		}

		cluster.SslOpts = sslOpts
	}

	// check and create test_keyspace
	if testOnly {
		cluster.Keyspace = dbconf.GetString("test_keyspace", DefaultTestKeyspace)
	} else {
		cluster.Keyspace = dbconf.GetString("keyspace", DefaultKeyspace)
	}

	// now point to the real keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, common.NewError(err)
	}
	session.SetPageSize(int(dbconf.GetInt32("page_size", DefaultPageSize)))

	blockedSubdocIds := conf.GetStringList("webconfig.blocked_subdoc_ids")
	encryptedSubdocIds := conf.GetStringList("webconfig.encrypted_subdoc_ids")
	stateCorrectionEnabled := conf.GetBoolean("webconfig.state_correction_enabled")
	lockRootDocumentEnabled := conf.GetBoolean("webconfig.lock_root_document_enabled")
	supplementaryPrecookEnabled := conf.GetBoolean("webconfig.supplementary_precook_enabled")
	supplementaryPrecookStateTTLDays := int(conf.GetInt32("webconfig.supplementary_precook_state_ttl_days", 7))

	return &CassandraClient{
		Session:                          session,
		ClusterConfig:                    cluster,
		AesCodec:                         codec,
		concurrentQueries:                make(chan bool, dbconf.GetInt32("concurrent_queries", 500)),
		localDc:                          localDc,
		blockedSubdocIds:                 blockedSubdocIds,
		encryptedSubdocIds:               encryptedSubdocIds,
		stateCorrectionEnabled:           stateCorrectionEnabled,
		lockRootDocumentEnabled:          lockRootDocumentEnabled,
		supplementaryPrecookEnabled:      supplementaryPrecookEnabled,
		supplementaryPrecookStateTTLDays: supplementaryPrecookStateTTLDays,
	}, nil
}

// loadCassandraTLSConfig loads TLS configuration for Cassandra connection.
// Returns a tls.Config with certificates loaded from the configuration.
// The function expects tls.{} block under the database driver config (cassandra or yugabyte).
func loadCassandraTLSConfig(dbconf *configuration.Config, dbdriver string) (*tls.Config, error) {
	// Check insecure_skip_verify flag first
	insecureSkipVerify := dbconf.GetBoolean("tls.insecure_skip_verify")

	// Load client certificates for mTLS if provided (optional when insecure_skip_verify is true)
	certFile := dbconf.GetString("tls.cert_file")
	keyFile := dbconf.GetString("tls.key_file")
	caCertFile := dbconf.GetString("tls.ca_cert_file")

	// Create TLS config for Cassandra connection with compatible cipher suite
	// Cassandra 3.11.x requires specific cipher suites that are disabled by default in newer Go crypto
	tlsConfig := &tls.Config{
		CipherSuites: []uint16{
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		},
		InsecureSkipVerify: insecureSkipVerify,
	}

	// When insecure_skip_verify is true and no cert files configured, skip loading certificates
	// This allows TLS without client authentication (server-only TLS)
	if insecureSkipVerify && (len(certFile) == 0 || len(keyFile) == 0) {
		// Insecure mode without client certificates - skip cert loading
		log.WithFields(log.Fields{
			"driver": dbdriver,
		}).Warn("Cassandra TLS enabled in insecure mode without client certificates")
	} else if len(certFile) > 0 && len(keyFile) > 0 {
		// Only validate cert files exist when verification is enabled
		if !insecureSkipVerify {
			// Validate certificate file exists
			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				return nil, fmt.Errorf("Cassandra TLS certificate file does not exist: %s", certFile)
			}

			// Validate key file exists
			if _, err := os.Stat(keyFile); os.IsNotExist(err) {
				return nil, fmt.Errorf("Cassandra TLS key file does not exist: %s", keyFile)
			}
		}

		// Load and parse the certificate and key
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load Cassandra TLS certificate and key from %s and %s: %v", certFile, keyFile, err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
		log.WithFields(log.Fields{
			"driver":    dbdriver,
			"cert_file": certFile,
			"key_file":  keyFile,
		}).Info("Loaded Cassandra TLS client certificate for mTLS")
	} else if len(certFile) > 0 || len(keyFile) > 0 {
		// Partial cert configuration detected - require both cert and key when verification is enabled
		if !insecureSkipVerify {
			return nil, fmt.Errorf("Cassandra TLS enabled with verification but incomplete certificate configuration (cert: %s, key: %s)", certFile, keyFile)
		}
	}

	// Load CA certificate if provided (optional when insecure_skip_verify is true)
	// When insecure_skip_verify is true and no CA file configured, skip loading CA cert
	// This allows TLS without server verification (insecure mode)
	if insecureSkipVerify && len(caCertFile) == 0 {
		// Insecure mode without CA cert - skip CA loading
		log.WithFields(log.Fields{
			"driver": dbdriver,
		}).Warn("Cassandra TLS enabled in insecure mode without CA certificate")
	} else if len(caCertFile) > 0 {
		// Only validate CA cert file exists when verification is enabled
		if !insecureSkipVerify {
			// Validate CA certificate file exists
			if _, err := os.Stat(caCertFile); os.IsNotExist(err) {
				return nil, fmt.Errorf("Cassandra TLS CA certificate file does not exist: %s", caCertFile)
			}
		}

		// Load CA certificate
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read Cassandra TLS CA certificate from %s: %v", caCertFile, err)
		}

		// Parse CA certificate
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse Cassandra TLS CA certificate from %s", caCertFile)
		}

		tlsConfig.RootCAs = caCertPool
		log.WithFields(log.Fields{
			"driver":       dbdriver,
			"ca_cert_file": caCertFile,
		}).Info("Loaded Cassandra TLS CA certificate for server verification")
	}

	if insecureSkipVerify {
		log.WithFields(log.Fields{
			"driver": dbdriver,
		}).Warn("Cassandra TLS certificate verification is disabled (insecure_skip_verify=true). This is insecure and should only be used for testing.")
	}

	log.WithFields(log.Fields{
		"driver":               dbdriver,
		"has_client_cert":      len(tlsConfig.Certificates) > 0,
		"has_ca_cert":          tlsConfig.RootCAs != nil,
		"insecure_skip_verify": insecureSkipVerify,
		"cipher_suites":        len(tlsConfig.CipherSuites),
	}).Info("TLS configuration loaded for Cassandra connection")

	return tlsConfig, nil
}

func (c *CassandraClient) Codec() *security.AesCodec {
	return c.AesCodec
}

func (c *CassandraClient) IsDbNotFound(err error) bool {
	return errors.Is(err, gocql.ErrNotFound)
}

func (c *CassandraClient) Close() error {
	c.Session.Close()
	return nil
}

func (c *CassandraClient) LocalDc() string {
	return c.localDc
}

func (c *CassandraClient) Metrics() *common.AppMetrics {
	return c.AppMetrics
}

func (c *CassandraClient) SetMetrics(m *common.AppMetrics) {
	c.AppMetrics = m
}

func (c *CassandraClient) IsMetricsEnabled() bool {
	return c.AppMetrics != nil
}

func (c *CassandraClient) BlockedSubdocIds() []string {
	return c.blockedSubdocIds
}

func (c *CassandraClient) SetBlockedSubdocIds(x []string) {
	c.blockedSubdocIds = x
}

func (c *CassandraClient) EncryptedSubdocIds() []string {
	return c.encryptedSubdocIds
}

func (c *CassandraClient) SetEncryptedSubdocIds(x []string) {
	c.encryptedSubdocIds = x
}

func (c *CassandraClient) StateCorrectionEnabled() bool {
	return c.stateCorrectionEnabled
}

func (c *CassandraClient) SetStateCorrectionEnabled(enabled bool) {
	c.stateCorrectionEnabled = enabled
}

func (c *CassandraClient) LockRootDocumentEnabled() bool {
	return c.lockRootDocumentEnabled
}

func (c *CassandraClient) SetLockRootDocumentEnabled(enabled bool) {
	c.lockRootDocumentEnabled = enabled
}

// TODO we hardcoded for now but it should be changed to be configurable
func (c *CassandraClient) IsEncryptedGroup(subdocId string) bool {
	return util.Contains(c.EncryptedSubdocIds(), subdocId)
}

func (c *CassandraClient) SetUp() error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// NOTE: CREATE cannot be used in a batch
	for _, t := range createTableStatements {
		if err := c.Query(t).Exec(); err != nil {
			return common.NewError(err)
		}
	}
	return nil
}

func (c *CassandraClient) TearDown() error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// NOTE: TRUNCATE cannot be used in a batch
	for t := range CassandraSchemas {
		if err := c.Query(fmt.Sprintf("TRUNCATE %v", t)).Exec(); err != nil {
			return common.NewError(err)
		}
	}
	return nil
}

// test dbclient by other modules
var (
	tdbclient *CassandraClient
	tcodec    *security.AesCodec
)

func GetTestCassandraClient(conf *configuration.Config, testOnly bool) (*CassandraClient, error) {
	if tdbclient != nil {
		return tdbclient, nil
	}

	var err error
	tdbclient, err = NewCassandraClient(conf, testOnly)
	if err != nil {
		return nil, common.NewError(err)
	}

	// Check if SKIP_TABLE_CREATION environment variable is set (case-insensitive)
	skipTableCreation := false
	if skipEnv, exists := os.LookupEnv("SKIP_TABLE_CREATION"); exists {
		skipTableCreation = strings.EqualFold(skipEnv, "true") || strings.EqualFold(skipEnv, "1") || strings.EqualFold(skipEnv, "yes")
	}

	if !skipTableCreation {
		err = tdbclient.SetUp()
		if err != nil {
			return nil, common.NewError(err)
		}
		err = tdbclient.TearDown()
		if err != nil {
			return nil, common.NewError(err)
		}
	}

	return tdbclient, nil
}

func (c *CassandraClient) SupplementaryPrecookEnabled() bool {
	return c.supplementaryPrecookEnabled
}

func (c *CassandraClient) SetSupplementaryPrecookEnabled(enabled bool) {
	c.supplementaryPrecookEnabled = enabled
}

func (c *CassandraClient) SupplementaryPrecookStateTTLDays() int {
	return c.supplementaryPrecookStateTTLDays
}

func (c *CassandraClient) SetSupplementaryPrecookStateTTLDays(days int) {
	c.supplementaryPrecookStateTTLDays = days
}
