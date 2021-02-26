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
package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-akka/configuration"
	log "github.com/sirupsen/logrus"
)

var (
	dbclient DatabaseClient
)

func TestMain(m *testing.M) {
	configFile := "config/sample_webconfig.conf"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = "../config/sample_webconfig.conf"
	}

	// configure the sqlite client
	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("ERROR: config file %v read error=%v\n", configFile, err)
		panic(err)
	}
	conf := configuration.ParseString(string(configBytes))

	dbclient, err = NewSqliteClient(conf, true)
	if err != nil {
		panic(err)
	}

	// start clean
	dbclient.SetUp()
	dbclient.TearDown()

	log.SetOutput(ioutil.Discard)

	returnCode := m.Run()

	// tear down
	// _ = suite.TearDown()

	os.Exit(returnCode)
}
