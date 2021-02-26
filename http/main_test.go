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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"
)

var (
	testConfigFile string
	sc             *common.ServerConfig
)

func ExecuteRequest(r *http.Request, handler http.Handler) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, r)
	return recorder
}

func TestMain(m *testing.M) {
	testConfigFile = "config/sample_webconfig.conf"
	if _, err := os.Stat(testConfigFile); os.IsNotExist(err) {
		testConfigFile = "../config/sample_webconfig.conf"
	}

	sid := os.Getenv("SAT_CLIENT_ID")
	if len(sid) == 0 {
		os.Setenv("SAT_CLIENT_ID", "foo")
	}

	sec := os.Getenv("SAT_CLIENT_SECRET")
	if len(sec) == 0 {
		os.Setenv("SAT_CLIENT_SECRET", "bar")
	}

	var err error
	sc, err = common.NewServerConfig(testConfigFile)
	if err != nil {
		panic(err)
	}
	server := NewWebconfigServer(sc, true, nil)

	// start clean
	server.SetUp()
	server.TearDown()

	log.SetOutput(ioutil.Discard)

	returnCode := m.Run()

	// tear down
	// _ = suite.TearDown()

	os.Exit(returnCode)
}
