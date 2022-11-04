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
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestCassandraClient(t *testing.T) {
	sc, err := common.GetTestServerConfig()
	assert.NilError(t, err)
	dbc, err := NewCassandraClient(sc.Config, true)
	assert.NilError(t, err)
	assert.Assert(t, dbc != nil)
	err = dbc.Close()
	assert.NilError(t, err)

	// test bool flags by random
	srcb := util.RandomBool()
	tdbclient.SetWifiSchemaMigrationEnabled(srcb)
	assert.Equal(t, tdbclient.WifiSchemaMigrationEnabled(), srcb)

	srcb = util.RandomBool()
	tdbclient.SetWifiSchemaV2Enabled(srcb)
	assert.Equal(t, tdbclient.WifiSchemaV2Enabled(), srcb)

	srcb = util.RandomBool()
	tdbclient.SetKeepTelcovoipOnFactoryResetEnabled(srcb)
	assert.Equal(t, tdbclient.KeepTelcovoipOnFactoryResetEnabled(), srcb)

	srcb = util.RandomBool()
	tdbclient.SetPartnerNoXdnsBitmapEnabled(srcb)
	assert.Equal(t, tdbclient.PartnerNoXdnsBitmapEnabled(), srcb)

	srcb = util.RandomBool()
	tdbclient.SetAppendLteProfilesEnabled(srcb)
	assert.Equal(t, tdbclient.AppendLteProfilesEnabled(), srcb)

	//  this need to be tested but finally set to be true
	srcb = false
	tdbclient.SetGroupConfigMigrationEnabled(srcb)
	assert.Equal(t, tdbclient.GroupConfigMigrationEnabled(), srcb)

	srcb = true
	tdbclient.SetGroupConfigMigrationEnabled(srcb)
	assert.Equal(t, tdbclient.GroupConfigMigrationEnabled(), srcb)

	_ = tdbclient.TraceEnabled()
	_ = tdbclient.LocalDc()
	assert.Assert(t, tdbclient.Codec() != nil)

	// -----
	odpKeyspace := util.GenerateRandomCpeMac()
	tdbclient.SetOdpKeyspace(odpKeyspace)
	assert.Equal(t, tdbclient.OdpKeyspace(), odpKeyspace)

	ksmap := tdbclient.KeyspaceSchemaMap()
	assert.Assert(t, len(ksmap) > 0)

	tableName := "xpc_group_config"
	columns, err := tdbclient.GetColumns(tableName)
	assert.NilError(t, err)
	assert.Assert(t, len(columns) >= 9)

	intColumns, err := tdbclient.GetIntColumns(tableName)
	assert.NilError(t, err)
	assert.Equal(t, len(intColumns), 2)

	tsColumns, err := tdbclient.GetTsColumns(tableName)
	assert.NilError(t, err)
	assert.Assert(t, len(tsColumns) >= 1)

	// XPC-15777
	// expect empty by default
	tgtSubdocIds := tdbclient.AutoMigrationExcludedSubdocIds()
	assert.Assert(t, len(tgtSubdocIds) == 0)

	srcSubdocIds := []string{"wan", "lan", "privatessid"}
	tdbclient.SetAutoMigrationExcludedSubdocIds(srcSubdocIds)
	tgtSubdocIds = tdbclient.AutoMigrationExcludedSubdocIds()
	assert.DeepEqual(t, srcSubdocIds, tgtSubdocIds)
}
