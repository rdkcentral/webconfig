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
	"time"

	"github.com/go-akka/configuration"
	"github.com/gocql/gocql"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/security"
)

const (
	ProtocolVersion               = 4
	DefaultKeyspace               = "xpc"
	DefaultOdpKeyspace            = "odp"
	DefaultTestKeyspace           = "test_webconfig"
	DisableInitialHostLookup      = false
	DefaultSleepTimeInMillisecond = 10
	DefaultConnections            = 2
	DefaultPageSize               = 50
)

// XPC-15293 if 'wifi_schema_v2_enabled'=true, v1.3 is also supported
type CassandraClient struct {
	db.BaseClient
	*gocql.Session
	*gocql.ClusterConfig
	*security.AesCodec
	*common.AppMetrics
	concurrentQueries                  chan bool
	groupConfigMigrationEnabled        bool
	tableColumnsMap                    map[string][]string
	tableIntColumnsMap                 map[string][]string
	tableTsColumnsMap                  map[string][]string
	traceEnabled                       bool
	localDc                            string
	odpKeyspace                        string
	keyspaceSchemaMap                  map[string]map[string]gocql.Type
	keepTelcovoipOnFactoryResetEnabled bool
	wifiSchemaV2Enabled                bool
	wifiSchemaMigrationEnabled         bool
	partnerNoXdnsBitmapEnabled         bool
	appendLteProfilesEnabled           bool
	syncFactoryResetRetainerEnabled    bool
	autoMigrationExcludedSubdocIds     []string
	factoryResetEnabled                bool
	blockedSubdocIds                   []string
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

// TODO need to reorganize this part
// keyspaceSchemaMap := map[string]map[string]gocql.Type{}

func GetColumnInfo(keyspaceSchemaMap map[string]map[string]gocql.Type, keyspace string) (map[string][]string, map[string][]string, map[string][]string) {
	tableColumnsMap := map[string][]string{}
	tableIntColumnsMap := map[string][]string{}
	tableTsColumnsMap := map[string][]string{}

	for tableName, tableSchema := range keyspaceSchemaMap {
		columns := []string{}
		intColumns := []string{}
		tsColumns := []string{}
		for columnName, columnType := range tableSchema {
			columns = append(columns, columnName)

			switch columnType {
			case gocql.TypeInt:
				intColumns = append(intColumns, columnName)
			case gocql.TypeTimestamp:
				tsColumns = append(tsColumns, columnName)
			}
		}
		tableColumnsMap[tableName] = columns
		tableIntColumnsMap[tableName] = intColumns
		tableTsColumnsMap[tableName] = tsColumns
	}
	return tableColumnsMap, tableIntColumnsMap, tableTsColumnsMap
}

func NewCassandraClient(conf *configuration.Config, testOnly bool) (*CassandraClient, error) {
	// if configured to not use db, then exit fast
	groupConfigMigrationEnabled := conf.GetBoolean("webconfig.database.cassandra.group_config_migration_enabled", false)
	wifiSchemaV2Enabled := conf.GetBoolean("webconfig.wifi_schema_v2_enabled", false)
	wifiSchemaMigrationEnabled := conf.GetBoolean("webconfig.wifi_schema_migration_enabled", false)
	partnerNoXdnsBitmapEnabled := conf.GetBoolean("webconfig.partner_no_xdns_bitmap_enabled", false)
	appendLteProfilesEnabled := conf.GetBoolean("webconfig.append_lte_profiles_enabled", false)

	// XPC-12206, note that the default is ON
	// XPC-12836, factory reset trigger is to be consolicated at xpc/sync.
	//            config webconfig.factory_reset_enabled should remain false until further notice
	//            config webconfig.keep_telcovoip_on_factory_reset_enabled ==> irrelevant
	keepTelcovoipOnFactoryResetEnabled := conf.GetBoolean("webconfig.keep_telcovoip_on_factory_reset_enabled", true)

	// XPC-15651
	syncFactoryResetRetainerEnabled := conf.GetBoolean("webconfig.sync_factory_reset_retainer_enabled", false)

	var codec *security.AesCodec
	var err error
	// build codec
	if testOnly {
		codec = security.NewTestCodec()
	} else {
		codec, err = security.NewAesCodec()
		if err != nil {
			return nil, common.NewError(err)
		}
	}

	// init
	hosts := conf.GetStringList("webconfig.database.cassandra.hosts")
	cluster := gocql.NewCluster(hosts...)

	cluster.Consistency = gocql.LocalQuorum
	cluster.ProtoVersion = ProtocolVersion
	cluster.DisableInitialHostLookup = DisableInitialHostLookup
	cluster.Timeout = time.Duration(conf.GetInt32("webconfig.database.cassandra.timeout_in_sec", 1)) * time.Second
	cluster.ConnectTimeout = time.Duration(conf.GetInt32("webconfig.database.cassandra.connect_timeout_in_sec", 1)) * time.Second
	cluster.NumConns = int(conf.GetInt32("webconfig.database.cassandra.connections", DefaultConnections))

	// XPC-8480
	cluster.RetryPolicy = &gocql.DowngradingConsistencyRetryPolicy{
		ConsistencyLevelsToTry: []gocql.Consistency{
			gocql.LocalQuorum,
			gocql.LocalOne,
			gocql.One,
		},
	}

	localDc := conf.GetString("webconfig.database.cassandra.local_dc")
	if len(localDc) > 0 {
		cluster.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(localDc)
	}

	var password string
	encryptedPassword := conf.GetString("webconfig.database.cassandra.encrypted_password")
	user := conf.GetString("webconfig.database.cassandra.user")
	isSslEnabled := conf.GetBoolean("webconfig.database.cassandra.is_ssl_enabled")

	// if the password is encrypted, we need to decrypt it
	if encryptedPassword != "" {
		password, err = codec.Decrypt(encryptedPassword)
		if err != nil {
			return nil, common.NewError(err)
		}

	} else {
		password = conf.GetString("webconfig.database.cassandra.password")
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
	var odpKeyspace string
	if testOnly {
		cluster.Keyspace = conf.GetString("webconfig.database.cassandra.test_keyspace", DefaultTestKeyspace)
		odpKeyspace = cluster.Keyspace
	} else {
		cluster.Keyspace = conf.GetString("webconfig.database.cassandra.keyspace", DefaultKeyspace)
		odpKeyspace = conf.GetString("webconfig.database.cassandra.odp_keyspace", DefaultOdpKeyspace)
	}

	// now point to the real keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, common.NewError(err)
	}
	session.SetPageSize(int(conf.GetInt32("webconfig.database.cassandra.page_size", DefaultPageSize)))

	// load schema/metadata from both xpc and odp keyspace
	keyspaceSchemaMap := map[string]map[string]gocql.Type{}
	if err = LoadSchemaMap(session, cluster.Keyspace, keyspaceSchemaMap); err != nil {
		return nil, common.NewError(err)
	}
	if err = LoadSchemaMap(session, odpKeyspace, keyspaceSchemaMap); err != nil {
		return nil, common.NewError(err)
	}

	// build the columns meta info
	tableColumnsMap, tableIntColumnsMap, tableTsColumnsMap := GetColumnInfo(keyspaceSchemaMap, cluster.Keyspace)

	traceEnabled := conf.GetBoolean("webconfig.database.cassandra.trace_enabled", false)
	factoryResetEnabled := conf.GetBoolean("webconfig.factory_reset_enabled", false)
	blockedSubdocIds := conf.GetStringList("webconfig.blocked_subdoc_ids")

	return &CassandraClient{
		Session:                            session,
		ClusterConfig:                      cluster,
		AesCodec:                           codec,
		concurrentQueries:                  make(chan bool, conf.GetInt32("webconfig.database.cassandra.concurrent_queries", 500)),
		groupConfigMigrationEnabled:        groupConfigMigrationEnabled,
		keyspaceSchemaMap:                  keyspaceSchemaMap,
		tableColumnsMap:                    tableColumnsMap,
		tableIntColumnsMap:                 tableIntColumnsMap,
		tableTsColumnsMap:                  tableTsColumnsMap,
		traceEnabled:                       traceEnabled,
		localDc:                            localDc,
		odpKeyspace:                        odpKeyspace,
		keepTelcovoipOnFactoryResetEnabled: keepTelcovoipOnFactoryResetEnabled,
		wifiSchemaV2Enabled:                wifiSchemaV2Enabled,
		wifiSchemaMigrationEnabled:         wifiSchemaMigrationEnabled,
		partnerNoXdnsBitmapEnabled:         partnerNoXdnsBitmapEnabled,
		appendLteProfilesEnabled:           appendLteProfilesEnabled,
		syncFactoryResetRetainerEnabled:    syncFactoryResetRetainerEnabled,
		factoryResetEnabled:                factoryResetEnabled,
		blockedSubdocIds:                   blockedSubdocIds,
	}, nil
}

func (c *CassandraClient) SetGroupConfigMigrationEnabled(groupConfigMigrationEnabled bool) {
	c.groupConfigMigrationEnabled = groupConfigMigrationEnabled
}

func (c *CassandraClient) GroupConfigMigrationEnabled() bool {
	return c.groupConfigMigrationEnabled
}

func (c *CassandraClient) SetWifiSchemaMigrationEnabled(wifiSchemaMigrationEnabled bool) {
	c.wifiSchemaMigrationEnabled = wifiSchemaMigrationEnabled
}

func (c *CassandraClient) WifiSchemaMigrationEnabled() bool {
	return c.wifiSchemaMigrationEnabled
}

// XPC-15293 if 'wifi_schema_v2_enabled'=true, v1.3 is also supported
func (c *CassandraClient) SetWifiSchemaV2Enabled(wifiSchemaV2Enabled bool) {
	c.wifiSchemaV2Enabled = wifiSchemaV2Enabled
}

func (c *CassandraClient) WifiSchemaV2Enabled() bool {
	return c.wifiSchemaV2Enabled
}

func (c *CassandraClient) GetColumns(tableName string) ([]string, error) {
	columns, ok := c.tableColumnsMap[tableName]
	if !ok {
		return nil, common.NewError(fmt.Errorf("No columns for table=%v", tableName))
	}
	return columns, nil
}

func (c *CassandraClient) GetIntColumns(tableName string) ([]string, error) {
	columns, ok := c.tableIntColumnsMap[tableName]
	if !ok {
		return nil, common.NewError(fmt.Errorf("No int columns data for table=%v", tableName))
	}
	return columns, nil
}

func (c *CassandraClient) GetTsColumns(tableName string) ([]string, error) {
	columns, ok := c.tableTsColumnsMap[tableName]
	if !ok {
		return nil, common.NewError(fmt.Errorf("No int columns data for table=%v", tableName))
	}
	return columns, nil
}

func (c *CassandraClient) TraceEnabled() bool {
	return c.traceEnabled
}

func (c *CassandraClient) LocalDc() string {
	return c.localDc
}

func (c *CassandraClient) Codec() *security.AesCodec {
	return c.AesCodec
}

func (c *CassandraClient) SetOdpKeyspace(odpKeyspace string) {
	c.odpKeyspace = odpKeyspace
}

func (c *CassandraClient) OdpKeyspace() string {
	return c.odpKeyspace
}

func (c *CassandraClient) IsDbNotFound(err error) bool {
	return errors.Is(err, gocql.ErrNotFound)
}

func (c *CassandraClient) Close() error {
	c.Session.Close()
	return nil
}

func LoadSchemaMap(s *gocql.Session, keyspace string, keyspaceSchemaMap map[string]map[string]gocql.Type) error {
	keyspaceMeta, err := s.KeyspaceMetadata(keyspace)
	if err != nil {
		return common.NewError(err)
	}

	for tableName, tableMeta := range keyspaceMeta.Tables {
		tableSchema := map[string]gocql.Type{}
		for columnName, columnMeta := range tableMeta.Columns {
			tableSchema[columnName] = columnMeta.Type.Type()
		}
		keyspaceSchemaMap[tableName] = tableSchema
	}
	return nil
}

func (c *CassandraClient) KeyspaceSchemaMap() map[string]map[string]gocql.Type {
	return c.keyspaceSchemaMap
}

// XPC-12206 keep telcovoip on factory reset
func (c *CassandraClient) SetKeepTelcovoipOnFactoryResetEnabled(keepTelcovoipOnFactoryResetEnabled bool) {
	c.keepTelcovoipOnFactoryResetEnabled = keepTelcovoipOnFactoryResetEnabled
}

func (c *CassandraClient) KeepTelcovoipOnFactoryResetEnabled() bool {
	return c.keepTelcovoipOnFactoryResetEnabled
}

// XPC-15030
func (c *CassandraClient) PartnerNoXdnsBitmapEnabled() bool {
	return c.partnerNoXdnsBitmapEnabled
}

func (c *CassandraClient) SetPartnerNoXdnsBitmapEnabled(enabled bool) {
	c.partnerNoXdnsBitmapEnabled = enabled
}

// XPC-14914
// TODO, still need this?
func (c *CassandraClient) SetMetrics(m *common.AppMetrics) {
	c.AppMetrics = m
}

func (c *CassandraClient) IsMetricsEnabled() bool {
	return c.AppMetrics != nil
}

// XPC-14586
func (c *CassandraClient) AppendLteProfilesEnabled() bool {
	return c.appendLteProfilesEnabled
}

func (c *CassandraClient) SetAppendLteProfilesEnabled(enabled bool) {
	c.appendLteProfilesEnabled = enabled
}

// XPC-15651
func (c *CassandraClient) SyncFactoryResetRetainerEnabled() bool {
	return c.syncFactoryResetRetainerEnabled
}

func (c *CassandraClient) SetSyncFactoryResetRetainerEnabled(enabled bool) {
	c.syncFactoryResetRetainerEnabled = enabled
}

// XPC-15777
func (c *CassandraClient) AutoMigrationExcludedSubdocIds() []string {
	return c.autoMigrationExcludedSubdocIds
}

func (c *CassandraClient) SetAutoMigrationExcludedSubdocIds(subdocIds []string) {
	c.autoMigrationExcludedSubdocIds = subdocIds
}

func (c *CassandraClient) FactoryResetEnabled() bool {
	return c.factoryResetEnabled
}

func (c *CassandraClient) SetFactoryResetEnabled(x bool) {
	c.factoryResetEnabled = x
}

func (c *CassandraClient) BlockedSubdocIds() []string {
	return c.blockedSubdocIds
}

func (c *CassandraClient) SetBlockedSubdocIds(x []string) {
	c.blockedSubdocIds = x
}

// TODO we hardcoded for now but it should be changed to be configurable
func (c *CassandraClient) IsEncryptedGroup(g string) bool {
	switch g {
	case "privatessid", "homessid", "telcovoip", "voiceservice":
		return true
	default:
		return false
	}
}
