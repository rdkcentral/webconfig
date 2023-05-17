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

CREATE KEYSPACE IF NOT EXISTS webconfig WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE webconfig;

CREATE TABLE root_document ( cpe_mac text PRIMARY KEY, bitmap bigint, firmware_version text, model_name text, partner_id text, query_params text, schema_version text, version text);

CREATE TABLE xpc_group_config ( cpe_mac text, group_id text, error_code int, error_details text, expiry timestamp, params text, payload blob, state int, updated_time timestamp, version text, PRIMARY KEY (cpe_mac, group_id)) WITH CLUSTERING ORDER BY (group_id ASC);

