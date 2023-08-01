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
	"io"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
)

var (
	sc *common.ServerConfig
)

func TestMain(m *testing.M) {
	var err error
	sc, err = common.GetTestServerConfig()
	if err != nil {
		panic(err)
	}

	tdbclient, err = GetTestSqliteClient(sc.Config, true)
	if err != nil {
		panic(err)
	}

	// TODO eval start clean
	// tdbclient.SetUp()
	// tdbclient.TearDown()

	log.SetOutput(io.Discard)

	tmetrics = common.NewMetrics(sc.Config)
	tdbclient.SetMetrics(tmetrics)

	returnCode := m.Run()

	// tear down
	// _ = suite.TearDown()

	os.Exit(returnCode)
}
