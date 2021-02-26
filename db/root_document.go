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

	"github.com/rdkcentral/webconfig/common"
)

func (c *SqliteClient) GetRootDocument(cpeMac string) (*common.RootDocument, error) {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	rows, err := c.Query(`SELECT version,bitmap FROM root_document WHERE cpe_mac=?`, cpeMac)
	if err != nil {
		return nil, common.NewError(err)
	}

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	var ns sql.NullString
	var ni sql.NullInt64
	err = rows.Scan(&ns, &ni)
	defer rows.Close()
	if err != nil {
		return nil, common.NewError(err)
	}

	var version string
	var bitmap int
	if ns.Valid {
		version = ns.String
	}
	if ni.Valid {
		bitmap = int(ni.Int64)
	}

	return common.NewRootDocument(version, bitmap), nil
}

func (c *SqliteClient) insertRootDocumentVersion(cpeMac string, version string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare(`INSERT INTO root_document(cpe_mac,version) VALUES(?,?)`)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac, version)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) updateRootDocumentVersion(cpeMac string, version string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare(`UPDATE root_document SET version=? WHERE cpe_mac=?`)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(version, cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) SetRootDocumentVersion(cpeMac string, version string) error {
	_, err := c.GetRootDocument(cpeMac)
	if err != nil {
		if c.IsDbNotFound(err) {
			return c.insertRootDocumentVersion(cpeMac, version)
		} else {
			// unexpected error
			return common.NewError(err)
		}
	}
	return c.updateRootDocumentVersion(cpeMac, version)
}

func (c *SqliteClient) insertRootDocumentBitmap(cpeMac string, bitmap int) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare(`INSERT INTO root_document(cpe_mac,bitmap) VALUES(?,?)`)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac, bitmap)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) updateRootDocumentBitmap(cpeMac string, bitmap int) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare(`UPDATE root_document SET bitmap=? WHERE cpe_mac=?`)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(bitmap, cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) SetRootDocumentBitmap(cpeMac string, bitmap int) error {
	_, err := c.GetRootDocument(cpeMac)
	if err != nil {
		if c.IsDbNotFound(err) {
			return c.insertRootDocumentBitmap(cpeMac, bitmap)
		} else {
			// unexpected error
			return common.NewError(err)
		}
	}
	return c.updateRootDocumentBitmap(cpeMac, bitmap)
}

func (c *SqliteClient) DeleteRootDocument(cpeMac string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare(`DELETE FROM root_document WHERE cpe_mac=?`)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) DeleteRootDocumentVersion(cpeMac string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	_, err := c.GetRootDocument(cpeMac)
	if err != nil {
		if c.IsDbNotFound(err) {
			return nil
		} else {
			// unexpected error
			return common.NewError(err)
		}
	}

	stmt, err := c.Prepare(`UPDATE root_document SET version=null WHERE cpe_mac=?`)
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}
