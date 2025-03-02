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
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-akka/configuration"
	"github.com/gocql/gocql"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/security"
	"github.com/rdkcentral/webconfig/util"
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
	concurrentQueries       chan bool
	localDc                 string
	blockedSubdocIds        []string
	encryptedSubdocIds      []string
	stateCorrectionEnabled  bool
	lockRootDocumentEnabled bool
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

func NewCassandraClient(conf *configuration.Config, testOnly bool, args ...interface{}) (*CassandraClient, error) {
	// ==== core objects BEGIN ====
	var codec *security.AesCodec
	// var err error
	var session *gocql.Session

	if len(args) > 0 {
		for _, itf := range args {
			switch ty := itf.(type) {
			case *security.AesCodec:
				codec = ty
			case *gocql.Session:
				session = ty
			}
		}
	}

	if codec == nil {
		ncodec, err := security.NewAesCodec(conf)
		if err != nil {
			return nil, common.NewError(err)
		}
		codec = ncodec
	}

	if session == nil {
		nsession, err := NewCassandraSession(conf, testOnly, codec)
		if err != nil {
			return nil, common.NewError(err)
		}
		session = nsession
	}
	// ==== core objects END   ====

	blockedSubdocIds := conf.GetStringList("webconfig.blocked_subdoc_ids")
	encryptedSubdocIds := conf.GetStringList("webconfig.encrypted_subdoc_ids")
	stateCorrectionEnabled := conf.GetBoolean("webconfig.state_correction_enabled")
	dbdriver := "cassandra"
	dbconf := conf.GetConfig("webconfig.database." + dbdriver)
	localDc := dbconf.GetString("local_dc")
	lockRootDocumentEnabled := conf.GetBoolean("webconfig.lock_root_document_enabled")

	return &CassandraClient{
		Session:                 session,
		AesCodec:                codec,
		concurrentQueries:       make(chan bool, dbconf.GetInt32("concurrent_queries", 500)),
		localDc:                 localDc,
		blockedSubdocIds:        blockedSubdocIds,
		encryptedSubdocIds:      encryptedSubdocIds,
		stateCorrectionEnabled:  stateCorrectionEnabled,
		lockRootDocumentEnabled: lockRootDocumentEnabled,
	}, nil
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
	fmt.Printf("is metrics enabled = %v\n", c.AppMetrics != nil)
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
	t0 := time.Now()
	defer func() {
		fmt.Printf("rdkw.Setup(): %v\n", time.Since(t0))
	}()

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
	t0 := time.Now()
	defer func() {
		fmt.Printf("rdkw.TearDown(): %v\n", time.Since(t0))
	}()

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

func NewCassandraSession(conf *configuration.Config, testOnly bool, codec *security.AesCodec) (*gocql.Session, error) {
	dbdriver := "cassandra"
	dbconf := conf.GetConfig("webconfig.database." + dbdriver)
	localDc := dbconf.GetString("local_dc")

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
		decrypted, err := codec.Decrypt(encryptedPassword)
		if err != nil {
			return nil, common.NewError(err)
		}
		password = decrypted
	} else {
		password = dbconf.GetString("password")
	}

	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: user,
		Password: password,
	}

	if isSslEnabled {
		sslOpts := &gocql.SslOptions{
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
	return session, nil
}
