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
	"github.com/go-akka/configuration"
	"github.com/gocql/gocql"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/security"
)

var (
	tsession *gocql.Session
)

func GetTestCassandraSession(conf *configuration.Config, testOnly bool) (*gocql.Session, error) {
	if tsession == nil {
		tdbclient, err := NewCassandraClient(conf, testOnly)
		if err != nil {
			return nil, common.NewError(err)
		}
		err = tdbclient.SetUp()
		if err != nil {
			return nil, common.NewError(err)
		}
		err = tdbclient.TearDown()
		if err != nil {
			return nil, common.NewError(err)
		}
		tsession = tdbclient.Session
	}

	return tsession, nil
}

func GetTestCassandraClient(conf *configuration.Config, testOnly bool) (*CassandraClient, error) {
	codec, err := security.GetTestCodec(conf)
	if err != nil {
		return nil, common.NewError(err)
	}

	session, err := GetTestCassandraSession(conf, testOnly)
	if err != nil {
		return nil, common.NewError(err)
	}

	dbclient, err := NewCassandraClient(conf, testOnly, codec, session)
	if err != nil {
		return nil, common.NewError(err)
	}
	return dbclient, nil
}
