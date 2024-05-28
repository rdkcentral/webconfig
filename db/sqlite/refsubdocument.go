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
	"fmt"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	_ "github.com/mattn/go-sqlite3"
)

func (c *SqliteClient) GetRefSubDocument(refId string) (*common.RefSubDocument, error) {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	rows, err := c.Query("SELECT payload,version FROM reference_document WHERE ref_id=?", refId)
	if err != nil {
		return nil, common.NewError(err)
	}

	var ns1 sql.NullString
	var b1 []byte

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	err = rows.Scan(&b1, &ns1)
	defer rows.Close()
	if err != nil {
		return nil, common.NewError(err)
	}

	var s1 *string
	if ns1.Valid {
		s1 = &(ns1.String)
	}

	refsubdoc := common.NewRefSubDocument(b1, s1)
	return refsubdoc, nil
}

func (c *SqliteClient) insertRefSubDocument(refId string, refsubdoc *common.RefSubDocument) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// build the statement and avoid unnecessary fields/columns
	columns := []string{"ref_id"}
	values := []interface{}{refId}
	if refsubdoc.Payload() != nil {
		columns = append(columns, "payload")
		values = append(values, refsubdoc.Payload())
	}
	if refsubdoc.Version() != nil {
		columns = append(columns, "version")
		values = append(values, refsubdoc.Version())
	}
	qstr := fmt.Sprintf("INSERT INTO reference_document(%v) VALUES(%v)", db.GetColumnsStr(columns), db.GetValuesStr(len(columns)))
	stmt, err := c.Prepare(qstr)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(values...)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) updateRefSubDocument(refId string, doc *common.RefSubDocument) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// build the statement and avoid unnecessary fields/columns
	columns := []string{}
	values := []interface{}{}
	if doc.Payload() != nil {
		columns = append(columns, "payload")
		values = append(values, doc.Payload())
	}
	if doc.Version() != nil {
		columns = append(columns, "version")
		values = append(values, doc.Version())
	}
	values = append(values, refId)
	qstr := fmt.Sprintf("UPDATE reference_document SET %v WHERE cpe_mac=? AND ref_id=?", db.GetSetColumnsStr(columns))
	stmt, err := c.Prepare(qstr)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(values...)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) SetRefSubDocument(refId string, refsubdoc *common.RefSubDocument) error {
	_, err := c.GetRefSubDocument(refId)
	if err != nil {
		if c.IsDbNotFound(err) {
			err1 := c.insertRefSubDocument(refId, refsubdoc)
			if err1 != nil {
				return common.NewError(err1)
			}
		} else {
			// unexpected error
			return common.NewError(err)
		}
	} else {
		// normal dbNotFound should not happen
		err = c.updateRefSubDocument(refId, refsubdoc)
		if err != nil {
			return common.NewError(err)
		}
	}

	return nil
}

func (c *SqliteClient) DeleteRefSubDocument(refId string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare("DELETE FROM reference_document WHERE ref_id=?")
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(refId)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}
