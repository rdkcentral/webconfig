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

// WARNING this function is planned to be called ONLY within SetUp(), so no
//
//	fetching from db CCR
func (c *CassandraClient) dropTables(tables []string) error {
	if len(tables) > 0 {
		// fmt.Printf("dropTables() tables=%v\n", tables)
		return nil
	}
	for _, t := range tables {
		if err := c.Query(fmt.Sprintf("DROP TABLE %v", t)).Exec(); err != nil {
			fmt.Printf("DROP TABLE %v\n", t)
			return common.NewError(err)
		}
	}
	return nil
}

// WARNING this function is planned to be called ONLY within SetUp(), so no
//
//	fetching from db CCR
func (c *CassandraClient) getOutOfSyncTables() ([]string, error) {
	if len(c.Keyspace) == 0 {
		return nil, fmt.Errorf("Empty keyspace in getOutOfSyncTables()")
	}

	keyspaceSchemaMap := c.KeyspaceSchemaMap()
	outOfSyncTables := []string{}

	// SetUp() will go through "createTableStatements" to do "create if not exists"
	// here we only try to find mismatches, so it is ok to start looping existing tables
	for tableName, tableSchema := range keyspaceSchemaMap {
		if schema, ok := ManagedXdpSchemas[tableName]; ok {
			// fail fast
			var matched bool
			if len(schema) != len(tableSchema) {
				matched = false
			} else {
				matched = true
				for column, ctype := range schema {
					if currType, ok := tableSchema[column]; ok {
						if currType != ctype {
							matched = false
							break
						}
					} else {
						matched = false
						break
					}
				}
			}

			// loop through expected
			if !matched {
				outOfSyncTables = append(outOfSyncTables, tableName)
			}
		}
	}

	return outOfSyncTables, nil
}

func (c *CassandraClient) SetUp() error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// %% for now detect and dropTable() are skipped
	// outOfSyncTables, err := c.getOutOfSyncTables()
	// if err != nil {
	// 	return common.NewError(err)
	// }
	// fmt.Fprintf(os.Stderr, "out of sync tables detected: %v\n", outOfSyncTables)

	// WARNING: my experiences showed that constantly dropping table is easy
	//          to cause "column family ID mismatch" error. Hence I decided
	//          to stop dropping tables here
	//err = c.dropTables(outOfSyncTables)
	//if err != nil {
	//    return common.NewError(err)
	//}

	// NOTE: CREATE cannot be used in a batch
	for _, t := range createTableStatements {
		if err := c.Query(t).Exec(); err != nil {
			fmt.Printf("error at t=%v\n", t)
			return common.NewError(err)
		}
	}
	return nil
}

func (c *CassandraClient) TearDown() error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// NOTE: TRUNCATE cannot be used in a batch
	for _, t := range AllTables {
		if err := c.Query(fmt.Sprintf("TRUNCATE %v", t)).Exec(); err != nil {
			return common.NewError(err)
		}
	}
	return nil
}
