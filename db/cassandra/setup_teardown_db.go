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

// Functions here are used to setup() and teardown() tables for unit test

import (
	"fmt"

	"github.com/go-akka/configuration"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/security"
)

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
	err = tdbclient.SetUp()
	if err != nil {
		return nil, common.NewError(err)
	}
	err = tdbclient.TearDown()
	if err != nil {
		return nil, common.NewError(err)
	}
	return tdbclient, nil
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
