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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
)

// shared.go:	err := c.Query(stmt, cpeMac).MapScan(dict)

func (c *CassandraClient) GetRootDocument(cpeMac string) (*common.RootDocument, error) {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	var rd common.RootDocument
	var tobj time.Time
	stmt := "SELECT bitmap,firmware_version,model_name,partner_id,schema_version,version,query_params,locked_till FROM root_document WHERE cpe_mac=?"
	err := c.Query(stmt, cpeMac).Scan(&rd.Bitmap, &rd.FirmwareVersion, &rd.ModelName, &rd.PartnerId, &rd.SchemaVersion, &rd.Version, &rd.QueryParams, &tobj)
	if err != nil {
		return nil, common.NewError(err)
	}
	if tobj.IsZero() {
		rd.LockedTill = 0
	} else {
		rd.LockedTill = int(tobj.UnixMilli())
	}
	return &rd, nil
}

// REMINDER this function is NOT identical to the UpdateRootDocument(), the columnMap(s) are different
func (c *CassandraClient) SetRootDocument(cpeMac string, rdoc *common.RootDocument) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	columns := []string{"cpe_mac"}
	values := []interface{}{cpeMac}
	columnMap := rdoc.NonEmptyColumnMap()
	for k, v := range columnMap {
		columns = append(columns, k)
		values = append(values, v)
	}

	stmt := fmt.Sprintf("INSERT INTO root_document(%v) VALUES(%v)", db.GetColumnsStr(columns), db.GetValuesStr(len(columns)))
	if err := c.Query(stmt, values...).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) DeleteRootDocument(cpeMac string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "DELETE FROM root_document WHERE cpe_mac = ?"
	if err := c.Query(stmt, cpeMac).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) SetRootDocumentVersion(cpeMac string, version string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "INSERT INTO root_document(cpe_mac,version) VALUES(?,?)"
	if err := c.Query(stmt, cpeMac, version).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) SetRootDocumentBitmap(cpeMac string, bitmap int) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "INSERT INTO root_document(cpe_mac,bitmap) VALUES(?,?)"
	if err := c.Query(stmt, cpeMac, int64(bitmap)).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) SetRootDocumentVersionBitmap(cpeMac string, version *string, bitmap *int64) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "INSERT INTO root_document(cpe_mac,version,bitmap) VALUES(?,?,?)"
	if err := c.Query(stmt, cpeMac, version, bitmap).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) GetRootDocumentVersionBitmap(cpeMac string, version *string, bitmap *int64) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "SELECT version,bitmap FROM root_document WHERE cpe_mac=?"
	if err := c.Query(stmt, cpeMac).Scan(version, bitmap); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) DeleteRootDocumentVersion(cpeMac string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "DELETE version FROM root_document WHERE cpe_mac = ?"
	if err := c.Query(stmt, cpeMac).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) DeleteRootDocumentBitmap(cpeMac string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "DELETE bitmap FROM root_document WHERE cpe_mac = ?"
	if err := c.Query(stmt, cpeMac).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) GetRootDocumentLabels(cpeMac string) (prometheus.Labels, error) {
	rdoc, err := c.GetRootDocument(cpeMac)
	if err != nil {
		if !c.IsDbNotFound(err) {
			return nil, common.NewError(err)
		}
		labels := prometheus.Labels{
			"model":     "unknown",
			"fwversion": "unknown",
		}
		return labels, nil
	}
	labels := prometheus.Labels{
		"model":     rdoc.ModelName,
		"fwversion": rdoc.FirmwareVersion,
	}
	return labels, nil
}
