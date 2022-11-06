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
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-akka/configuration"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
)

const (
	defaultSqliteDbFile        = "/app/db/webconfigcommon.db"
	defaultSqliteTestDbFile    = "/app/db/test_webconfigcommon.db"
	defaultDbConcurrentQueries = 10
)

var (
	tdbclient *SqliteClient
	tmetrics  *common.AppMetrics
)

type SqliteClient struct {
	db.BaseClient
	*sql.DB
	*common.AppMetrics
	concurrentQueries chan bool
}

func NewSqliteClient(conf *configuration.Config, testOnly bool) (*SqliteClient, error) {
	// check and create test_keyspace
	var dbfile string
	if testOnly {
		dbfile = conf.GetString("webconfig.database.sqlite.unittest_db_file", defaultSqliteTestDbFile)
	} else {
		dbfile = conf.GetString("webconfig.database.sqlite.db_file", defaultSqliteDbFile)
	}

	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		return nil, common.NewError(err)
	}

	return &SqliteClient{
		DB:                db,
		concurrentQueries: make(chan bool, conf.GetInt32("webconfig.database.sqlite.concurrent_queries", defaultDbConcurrentQueries)),
	}, nil
}

func (c *SqliteClient) SetUp() error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// NOTE: CREATE cannot be used in a batch
	for _, t := range SqliteCreateTableStatements {
		stmt, err := c.Prepare(t)
		if err != nil {
			return common.NewError(err)
		}

		if _, err := stmt.Exec(); err != nil {
			return common.NewError(err)
		}

	}
	return nil
}

func (c *SqliteClient) TearDown() error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	for _, t := range SqliteAllTables {
		stmt, err := c.Prepare(fmt.Sprintf("DROP TABLE %v", t))
		if err != nil {
			return common.NewError(err)
		}

		if _, err := stmt.Exec(); err != nil {
			return common.NewError(err)
		}
	}
	return nil
}

func (c *SqliteClient) IsDbNotFound(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return false
}

func (c *SqliteClient) SetMetrics(m *common.AppMetrics) {
	c.AppMetrics = m
}

func (c *SqliteClient) IsMetricsEnabled() bool {
	if c.AppMetrics == nil {
		return false
	}
	return true
}

func GetTestSqliteClient(conf *configuration.Config, testOnly bool) (*SqliteClient, error) {
	if tdbclient != nil {
		return tdbclient, nil
	}
	var err error
	tdbclient, err = NewSqliteClient(conf, testOnly)
	if err != nil {
		return nil, common.NewError(err)
	}

	// need to do it this way to sure we have correct schema but empty tables
	if err = tdbclient.SetUp(); err != nil {
		return nil, common.NewError(err)
	}
	if err = tdbclient.TearDown(); err != nil {
		return nil, common.NewError(err)
	}
	if err = tdbclient.SetUp(); err != nil {
		return nil, common.NewError(err)
	}

	return tdbclient, nil
}
