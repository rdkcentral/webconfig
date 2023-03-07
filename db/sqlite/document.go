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

	_ "github.com/mattn/go-sqlite3"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
)

func (c *SqliteClient) GetSubDocument(cpeMac string, groupId string) (*common.SubDocument, error) {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	rows, err := c.Query("SELECT payload,state,updated_time,version,error_code,error_details FROM xpc_group_config WHERE cpe_mac=? AND group_id=?", cpeMac, groupId)
	if err != nil {
		return nil, common.NewError(err)
	}

	var ns1, ns2 sql.NullString
	var b1 []byte
	var nt1 sql.NullTime
	var ni1, ni2 sql.NullInt64

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	err = rows.Scan(&b1, &ni1, &nt1, &ns1, &ni2, &ns2)
	defer rows.Close()
	if err != nil {
		return nil, common.NewError(err)
	}

	var s1, s2 *string
	var i1, i2 *int
	var ts *int
	if ns1.Valid {
		s1 = &(ns1.String)
	}
	if ns2.Valid {
		s2 = &(ns2.String)
	}
	if nt1.Valid {
		t1 := nt1.Time
		tt := int(t1.UnixNano() / 1000000)
		ts = &tt
	}
	if ni1.Valid {
		ii := int(ni1.Int64)
		i1 = &ii
	}
	if ni2.Valid {
		ii := int(ni2.Int64)
		i2 = &ii
	}

	doc := common.NewSubDocument(b1, s1, i1, ts, i2, s2)
	return doc, nil
}

func (c *SqliteClient) insertSubDocument(cpeMac string, groupId string, doc *common.SubDocument) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	// build the statement and avoid unnecessary fields/columns
	columns := []string{"cpe_mac", "group_id"}
	values := []interface{}{cpeMac, groupId}
	if doc.Payload() != nil {
		columns = append(columns, "payload")
		values = append(values, doc.Payload())
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
	qstr := fmt.Sprintf("INSERT INTO xpc_group_config(%v) VALUES(%v)", db.GetColumnsStr(columns), db.GetValuesStr(len(columns)))
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

func (c *SqliteClient) updateSubDocument(cpeMac string, groupId string, doc *common.SubDocument) error {
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
	qstr := fmt.Sprintf("UPDATE xpc_group_config SET %v WHERE cpe_mac=? AND group_id=?", db.GetSetColumnsStr(columns))
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

func (c *SqliteClient) SetSubDocument(cpeMac string, groupId string, doc *common.SubDocument, vargs ...interface{}) error {
	var oldState int
	client := "default"
	for _, varg := range vargs {
		switch ty := varg.(type) {
		case int:
			oldState = ty
		case string:
			client = ty
		}
	}

	_, err := c.GetSubDocument(cpeMac, groupId)
	if err != nil {
		if c.IsDbNotFound(err) {
			err1 := c.insertSubDocument(cpeMac, groupId, doc)
			if err1 != nil {
				return common.NewError(err1)
			}
		} else {
			// unexpected error
			return common.NewError(err)
		}
	} else {
		// normal dbNotFound should not happen
		err = c.updateSubDocument(cpeMac, groupId, doc)
		if err != nil {
			return common.NewError(err)
		}
	}

	// update state metrics
	if c.IsMetricsEnabled() {
		if doc.State() != nil {
			c.UpdateStateMetrics(oldState, *doc.State(), groupId, client)
		}
	}
	return nil
}

func (c *SqliteClient) DeleteSubDocument(cpeMac string, groupId string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare("DELETE FROM xpc_group_config WHERE cpe_mac=? AND group_id=?")
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac, groupId)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) DeleteDocument(cpeMac string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare("DELETE FROM xpc_group_config WHERE cpe_mac=?")
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) GetDocument(cpeMac string, args ...bool) (*common.Document, error) {
	Document := common.NewDocument(nil)

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()
	// ns0,    b1,     ni1,  nt1,        ns1,     nil2      ns2
	rows, err := c.Query("SELECT group_id,payload,state,updated_time,version,error_code,error_details FROM xpc_group_config WHERE cpe_mac=?", cpeMac)
	if err != nil {
		return nil, common.NewError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var ns0, ns1, ns2 sql.NullString
		var b1 []byte
		var nt1 sql.NullTime
		var ni1, ni2 sql.NullInt64

		err = rows.Scan(&ns0, &b1, &ni1, &nt1, &ns1, &ni2, &ns2)
		if err != nil {
			return nil, common.NewError(err)
		}

		var s1, s2 *string
		var groupId string
		var i1, i2 *int
		var ts *int

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
			tt := int(t1.UnixNano() / 1000000)
			ts = &tt
		}
		if ni1.Valid {
			ii := int(ni1.Int64)
			i1 = &ii
		}
		if ni2.Valid {
			ii := int(ni2.Int64)
			i2 = &ii
		}

		doc := common.NewSubDocument(b1, s1, i1, ts, i2, s2)
		Document.SetSubDocument(groupId, doc)
	}

	if Document.Length() == 0 {
		return Document, common.NewError(sql.ErrNoRows)
	} else {
		return Document, nil
	}
}
