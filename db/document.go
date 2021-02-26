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
package db

import (
	"database/sql"
	"fmt"

	"github.com/rdkcentral/webconfig/common"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func (c *SqliteClient) GetDocument(cpeMac string, groupId string, fields log.Fields) (*common.Document, error) {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	rows, err := c.Query(`SELECT params,payload,state,updated_time,version FROM xpc_group_config WHERE cpe_mac=? AND group_id=?`, cpeMac, groupId)
	if err != nil {
		return nil, common.NewError(err)
	}

	var ns1, ns2 sql.NullString
	var b1 []byte
	var nt1 sql.NullTime
	var ni1 sql.NullInt64

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	err = rows.Scan(&ns1, &b1, &ni1, &nt1, &ns2)
	defer rows.Close()
	if err != nil {
		return nil, common.NewError(err)
	}

	var s1, s2 *string
	var i1 *int
	var ts *int64
	if ns1.Valid {
		s1 = &(ns1.String)
	}
	if ns2.Valid {
		s2 = &(ns2.String)
	}
	if nt1.Valid {
		t1 := nt1.Time
		tt := t1.UnixNano() / 1000000
		ts = &tt
	}
	if ni1.Valid {
		ii := int(ni1.Int64)
		i1 = &ii
	}

	doc := common.NewDocument(b1, s1, s2, i1, ts)
	return doc, nil
}

func (c *SqliteClient) insertDocument(cpeMac string, groupId string, doc *common.Document, fields log.Fields) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// build the statement and avoid unnecessary fields/columns
	columns := []string{"cpe_mac", "group_id"}
	values := []interface{}{cpeMac, groupId}
	if doc.Bytes() != nil {
		columns = append(columns, "payload")
		values = append(values, doc.Bytes())
	}
	if doc.Params() != nil {
		columns = append(columns, "params")
		values = append(values, doc.Params())
	}
	if doc.Version() != nil {
		columns = append(columns, "version")
		values = append(values, doc.Version())
	}
	if doc.State() != nil {
		columns = append(columns, "state")
		values = append(values, doc.State())
	}
	if doc.UpdatedTime() != nil {
		columns = append(columns, "updated_time")
		values = append(values, doc.UpdatedTime())
	}
	qstr := fmt.Sprintf(`INSERT INTO xpc_group_config(%v) VALUES(%v)`, GetColumnsStr(columns), GetValuesStr(len(columns)))
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

func (c *SqliteClient) updateDocument(cpeMac string, groupId string, doc *common.Document, fields log.Fields) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// build the statement and avoid unnecessary fields/columns
	columns := []string{}
	values := []interface{}{}
	if doc.Bytes() != nil {
		columns = append(columns, "payload")
		values = append(values, doc.Bytes())
	}
	if doc.Params() != nil {
		columns = append(columns, "params")
		values = append(values, doc.Params())
	}
	if doc.Version() != nil {
		columns = append(columns, "version")
		values = append(values, doc.Version())
	}
	if doc.State() != nil {
		columns = append(columns, "state")
		values = append(values, doc.State())
	}
	if doc.UpdatedTime() != nil {
		columns = append(columns, "updated_time")
		values = append(values, doc.UpdatedTime())
	}
	values = append(values, cpeMac)
	values = append(values, groupId)
	qstr := fmt.Sprintf(`UPDATE xpc_group_config SET %v WHERE cpe_mac=? AND group_id=?`, GetSetColumnsStr(columns))
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

func (c *SqliteClient) SetDocument(cpeMac string, groupId string, doc *common.Document, fields log.Fields) error {
	_, err := c.GetDocument(cpeMac, groupId, fields)
	if err != nil {
		if c.IsDbNotFound(err) {
			return c.insertDocument(cpeMac, groupId, doc, fields)
		} else {
			// unexpected error
			return common.NewError(err)
		}
	}
	return c.updateDocument(cpeMac, groupId, doc, fields)
}

func (c *SqliteClient) DeleteDocument(cpeMac string, groupId string, fields log.Fields) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare(`DELETE FROM xpc_group_config WHERE cpe_mac=? AND group_id=?`)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac, groupId)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) DeleteFullDocument(cpeMac string, fields log.Fields) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare(`DELETE FROM xpc_group_config WHERE cpe_mac=?`)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) GetFolder(cpeMac string, fields log.Fields) (*common.Folder, error) {
	folder := common.NewFolder()

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	rows, err := c.Query(`SELECT group_id,params,payload,state,updated_time,version FROM xpc_group_config WHERE cpe_mac=?`, cpeMac)
	if err != nil {
		return nil, common.NewError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var ns0, ns1, ns2 sql.NullString
		var b1 []byte
		var nt1 sql.NullTime
		var ni1 sql.NullInt64

		err = rows.Scan(&ns0, &ns1, &b1, &ni1, &nt1, &ns2)
		if err != nil {
			return nil, common.NewError(err)
		}

		var s1, s2 *string
		var groupId string
		var i1 *int
		var ts *int64

		if ns0.Valid {
			groupId = ns0.String
		}
		if ns1.Valid {
			s1 = &(ns1.String)
		}
		if ns2.Valid {
			s2 = &(ns2.String)
		}
		if nt1.Valid {
			t1 := nt1.Time
			tt := t1.UnixNano() / 1000000
			ts = &tt
		}
		if ni1.Valid {
			ii := int(ni1.Int64)
			i1 = &ii
		}

		doc := common.NewDocument(b1, s1, s2, i1, ts)
		folder.SetDocument(groupId, doc)
	}

	if folder.Length() == 0 {
		return folder, common.NewError(sql.ErrNoRows)
	} else {
		return folder, nil
	}
}
