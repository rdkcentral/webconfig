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
	"gotest.tools/assert"
)

func TestCassandraClient(t *testing.T) {
	sc, err := common.GetTestServerConfig()
	assert.NilError(t, err)
	dbc, err := GetTestCassandraClient(sc.Config, true)
	assert.NilError(t, err)
	assert.Assert(t, dbc != nil)

	_ = tdbclient.LocalDc()
	assert.Assert(t, tdbclient.Codec() != nil)

	// XPC-15777
	// expect empty by default
	tgtSubdocIds := tdbclient.EncryptedSubdocIds()
	assert.Assert(t, len(tgtSubdocIds) == 0)
}

func TestGetConfig(t *testing.T) {
	configFile := "../../config/sample_webconfigcommon.conf"
	sc, err := common.GetTestServerConfig(configFile)
	assert.NilError(t, err)

	subConfig := sc.Config.GetConfig("webconfig.database.cassandra")
	x := subConfig.GetString("keyspace")
	assert.Equal(t, x, "xpc")

	subConfig = sc.Config.GetConfig("webconfig.database.yugabyte")
	y := subConfig.GetString("keyspace")
	assert.Equal(t, y, "yugabytedb")
}
