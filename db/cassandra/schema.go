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
	"github.com/gocql/gocql"
)

var (
	createTableStatements = []string{
		`CREATE TABLE IF NOT EXISTS xpc_group_config (
    cpe_mac text,
    group_id text,
    error_code int,
    error_details text,
    expiry timestamp,
    payload blob,
    state int,
    updated_time timestamp,
    version text,
    PRIMARY KEY (cpe_mac, group_id)
)`,
		`CREATE TABLE IF NOT EXISTS root_document (
    cpe_mac text PRIMARY KEY,
    bitmap bigint,
    firmware_version text,
    model_name text,
    partner_id text,
    route text,
    schema_version text,
    version text
)`,
	}

	CassandraSchemas = map[string]map[string]gocql.Type{
		"xpc_group_config": {
			"cpe_mac":       gocql.TypeText,
			"group_id":      gocql.TypeText,
			"error_code":    gocql.TypeInt,
			"error_details": gocql.TypeText,
			"expiry":        gocql.TypeTimestamp,
			"payload":       gocql.TypeBlob,
			"state":         gocql.TypeInt,
			"updated_time":  gocql.TypeTimestamp,
			"version":       gocql.TypeText,
		},
		"root_document": {
			"cpe_mac":          gocql.TypeText,
			"bitmap":           gocql.TypeBigInt,
			"firmware_version": gocql.TypeText,
			"model_name":       gocql.TypeText,
			"partner_id":       gocql.TypeText,
			"route":            gocql.TypeText,
			"schema_version":   gocql.TypeText,
			"version":          gocql.TypeText,
		},
	}
)
