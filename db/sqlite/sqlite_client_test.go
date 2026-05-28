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
package sqlite

import (
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"gotest.tools/assert"
)

func TestSqliteClient(t *testing.T) {
	configFile := "../../config/sample_webconfig.conf"
	sc, err := common.GetTestServerConfig(configFile)

	assert.NilError(t, err)
	dbc, err := GetTestSqliteClient(sc.Config, true)
	assert.NilError(t, err)
	assert.Assert(t, dbc != nil)

	// state correction flag
	enabled := true
	tdbclient.SetStateCorrectionEnabled(enabled)
	assert.Equal(t, tdbclient.StateCorrectionEnabled(), enabled)
	enabled = false
	tdbclient.SetStateCorrectionEnabled(enabled)
	assert.Equal(t, tdbclient.StateCorrectionEnabled(), enabled)

	// lock root_document flag
	enabled = true
	tdbclient.SetLockRootDocumentEnabled(enabled)
	assert.Equal(t, tdbclient.LockRootDocumentEnabled(), enabled)
	enabled = false
	tdbclient.SetLockRootDocumentEnabled(enabled)
	assert.Equal(t, tdbclient.LockRootDocumentEnabled(), enabled)
}
