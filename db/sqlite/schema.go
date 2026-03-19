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
	"fmt"
	"regexp"
	"strings"
	"unicode"
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
    expiry timestamp,
    PRIMARY KEY (cpe_mac, group_id)
)`,
		`CREATE TABLE IF NOT EXISTS root_document (
    cpe_mac text PRIMARY KEY,
    bitmap bigint,
    customer_type text,
    firmware_version text,
    locked_till timestamp,
    model_name text,
    partner_id text,
    product_class text,
    query_params text,
    route text,
    schema_version,
    version text
)`,
		`CREATE TABLE IF NOT EXISTS reference_document (
    ref_id text PRIMARY KEY,
    payload blob,
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

// parseCreateTable extracts the table name and a map of column-name→type from a
// "CREATE TABLE IF NOT EXISTS" DDL string. It is used by SyncSchema to diff
// expected columns against what PRAGMA table_info reports.
func parseCreateTable(stmt string) (string, map[string]string, error) {
	reTableName := regexp.MustCompile(`(?i)CREATE TABLE IF NOT EXISTS (\w+)`)
	m := reTableName.FindStringSubmatch(stmt)
	if len(m) < 2 {
		return "", nil, fmt.Errorf("parseCreateTable: cannot find table name in DDL")
	}
	tableName := m[1]

	colDefs := make(map[string]string)
	for _, rawLine := range strings.Split(stmt, "\n") {
		line := strings.TrimSpace(rawLine)
		line = strings.TrimRight(line, ",")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		upper := strings.ToUpper(line)
		// skip the CREATE TABLE header, lone parens, and table-level constraints
		if strings.HasPrefix(upper, "CREATE") ||
			line == "(" || line == ")" ||
			strings.HasPrefix(upper, "PRIMARY") ||
			strings.HasPrefix(upper, "UNIQUE") ||
			strings.HasPrefix(upper, "FOREIGN") ||
			strings.HasPrefix(upper, "CHECK") ||
			strings.HasPrefix(upper, "CONSTRAINT") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		colName := parts[0]
		if !isSQLiteIdentifier(colName) {
			continue
		}
		var colType string
		if len(parts) > 1 {
			t := parts[1]
			u := strings.ToUpper(t)
			if u != "PRIMARY" && u != "NOT" && u != "UNIQUE" && u != "REFERENCES" {
				colType = t
			}
		}
		colDefs[colName] = colType
	}
	return tableName, colDefs, nil
}

// isSQLiteIdentifier reports whether s is a valid plain SQL identifier
// (ASCII letters, digits, and underscores only).
func isSQLiteIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return false
		}
	}
	return true
}
