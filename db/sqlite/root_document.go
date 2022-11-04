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

	"github.com/rdkcentral/webconfig/common"
)

func (c *SqliteClient) GetRootDocument(cpeMac string) (*common.RootDocument, error) {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	rows, err := c.Query("SELECT bitmap,firmware_version,model_name,partner_id,schema_version,version FROM root_document WHERE cpe_mac=?", cpeMac)
	if err != nil {
		return nil, common.NewError(err)
	}

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	var ni sql.NullInt64
	var ns1, ns2, ns3, ns4, ns5 sql.NullString
	err = rows.Scan(&ni, &ns1, &ns2, &ns3, &ns4, &ns5)
	defer rows.Close()
	if err != nil {
		return nil, common.NewError(err)
	}

	var bitmap int
	if ni.Valid {
		bitmap = int(ni.Int64)
	}

	var firmware_version, model_name, partner_id, schema_version, version string
	if ns1.Valid {
		firmware_version = ns1.String
	}
	if ns2.Valid {
		model_name = ns2.String
	}
	if ns3.Valid {
		partner_id = ns3.String
	}
	if ns4.Valid {
		schema_version = ns4.String
	}
	if ns5.Valid {
		version = ns5.String
	}

	return common.NewRootDocument(bitmap, firmware_version, model_name, partner_id, schema_version, version), nil
}

func (c *SqliteClient) insertRootDocumentVersion(cpeMac, version string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare("INSERT INTO root_document(cpe_mac,version) VALUES(?,?)")
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac, version)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) updateRootDocumentVersion(cpeMac, version string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare("UPDATE root_document SET version=? WHERE cpe_mac=?")
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(version, cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) SetRootDocumentVersion(cpeMac, version string) error {
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

	stmt, err := c.Prepare("INSERT INTO root_document(cpe_mac,bitmap) VALUES(?,?)")
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

	stmt, err := c.Prepare("UPDATE root_document SET bitmap=? WHERE cpe_mac=?")
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

	stmt, err := c.Prepare("DELETE FROM root_document WHERE cpe_mac=?")
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

	stmt, err := c.Prepare("UPDATE root_document SET version=null WHERE cpe_mac=?")
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

// 11111111111111111 batman
func (c *SqliteClient) insertRootDocument(cpeMac string, rd *common.RootDocument) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare("INSERT INTO root_document(cpe_mac,bitmap,firmware_version,model_name,partner_id,schema_version,version) VALUES(?,?,?,?,?,?,?)")
	if err != nil {
		return common.NewError(err)
	}

	_, err = stmt.Exec(cpeMac, rd.Bitmap, rd.FirmwareVersion, rd.ModelName, rd.PartnerId, rd.SchemaVersion, rd.Version)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *SqliteClient) updateRootDocument(cpeMac string, rd *common.RootDocument) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt, err := c.Prepare("UPDATE root_document SET bitmap=?,firmware_version=?,model_name=?,partner_id=?,schema_version=?,version=?  WHERE cpe_mac=?")
	if err != nil {
		return common.NewError(err)
	}
	_, err = stmt.Exec(rd.Bitmap, rd.FirmwareVersion, rd.ModelName, rd.PartnerId, rd.SchemaVersion, rd.Version, cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

// rows, err := c.Query("SELECT bitmap,firmware_version,model_name,partner_id,schema_version,version FROM root_document WHERE cpe_mac=?", cpeMac)
// 22222222222222222 batman

// simple implementation, could optimize if needed
func (c *SqliteClient) SetRootDocument(cpeMac string, inRootdoc *common.RootDocument) error {
	rootdoc, err := c.GetRootDocument(cpeMac)
	if err != nil {
		if !c.IsDbNotFound(err) {
			return common.NewError(err)
		}
		// db not found, create a new record
		if err = c.insertRootDocument(cpeMac, inRootdoc); err != nil {
			return common.NewError(err)
		}
		return nil
	}
	rootdoc.Update(inRootdoc)

	if err := c.updateRootDocument(cpeMac, rootdoc); err != nil {
		return common.NewError(err)
	}
	return nil
}

/*
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
*/

// TODO this needs to include other info in the headers
// func (c *SqliteClient) UpdateRootDocument(cpeMac string, rd *common.RootDocument) error {
// 	columnMap := rd.ChangedColumnMap()
// 	if _, ok := columnMap["version"]; ok {
// 		err := c.SetRootDocumentVersion(cpeMac, rd.Version())
// 		if err != nil {
// 			return common.NewError(err)
// 		}
// 	}

// 	if _, ok := columnMap["bitmap"]; ok {
// 		err := c.SetRootDocumentBitmap(cpeMac, rd.Bitmap())
// 		if err != nil {
// 			return common.NewError(err)
// 		}
// 	}
// 	return nil
// }
