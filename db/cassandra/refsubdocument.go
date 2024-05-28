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
	"fmt"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/gocql/gocql"
)

func (c *CassandraClient) GetRefSubDocument(refId string) (*common.RefSubDocument, error) {
	var payload []byte
	var version string

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "SELECT payload,version FROM reference_document WHERE ref_id=?"
	if err := c.Query(stmt, refId).Scan(&payload, &version); err != nil {
		return nil, common.NewError(err)
	}

	if len(payload) == 0 {
		return nil, common.NewError(gocql.ErrNotFound)
	}

	refsubdoc := common.NewRefSubDocument(payload, &version)
	return refsubdoc, nil
}

func (c *CassandraClient) SetRefSubDocument(refId string, refsubdoc *common.RefSubDocument) (fnerr error) {
	// build the statement and avoid unnecessary fields/columns
	columns := []string{"ref_id"}
	values := []interface{}{refId}
	if refsubdoc.Payload() != nil && len(refsubdoc.Payload()) > 0 {
		columns = append(columns, "payload")
		values = append(values, refsubdoc.Payload())
	}

	if refsubdoc.Version() != nil {
		columns = append(columns, "version")
		values = append(values, refsubdoc.Version())
	}
	stmt := fmt.Sprintf("INSERT INTO reference_document(%v) VALUES(%v)", db.GetColumnsStr(columns), db.GetValuesStr(len(columns)))

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	if err := c.Query(stmt, values...).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) DeleteRefSubDocument(refId string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "DELETE FROM reference_document WHERE ref_id=?"
	if err := c.Query(stmt, refId).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}
