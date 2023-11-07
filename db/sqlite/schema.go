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
	"regexp"
)

const (
	CreateKeyspaceStatement = `CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '1'}`
	regexPattern            = `^CREATE TABLE IF NOT EXISTS (?P<tablename>\w+) \(`
)

var (
	SqliteAllTables             = []string{}
	SqliteCreateTableStatements = []string{
		`CREATE TABLE IF NOT EXISTS xpc_group_config (
    cpe_mac text NOT NULL,
    group_id text NOT NULL,
    params text,
    updated_time timestamp,
    version text,
    payload blob,
    state int,
    error_code int,
    error_details text,
    PRIMARY KEY (cpe_mac, group_id)
)`,
		`CREATE TABLE IF NOT EXISTS root_document (
    cpe_mac text PRIMARY KEY,
    bitmap bigint,
    firmware_version text,
    model_name text,
    partner_id text,
    route text,
    schema_version,
    version text
)`,
	}
)

func init() {
	tbExp := regexp.MustCompile(regexPattern)

	for _, x := range SqliteCreateTableStatements {
		match := tbExp.FindStringSubmatch(x)
		if len(match) > 1 {
			SqliteAllTables = append(SqliteAllTables, match[1])
		}
	}
}
