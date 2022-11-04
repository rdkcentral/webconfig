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

	"github.com/gocql/gocql"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
)

/*
	stmt := "SELECT version,bitmap FROM root_document WHERE cpe_mac=?"
	if err := c.Query(stmt, cpeMac).Scan(version, bitmap); err != nil {
		return common.NewError(err)
	}
	return nil
type SubDocument struct {
    payload        []byte
    params       *string
    version      *string
    state        *int
    updatedTime  *int
    errorCode    *int
    errorDetails *string
}
*/

func (c *CassandraClient) GetSubDocument(cpeMac string, groupId string) (*common.SubDocument, error) {
	var err error
	var payload []byte
	var version, errorDetails string
	var state, errorCode int
	var updatedTime time.Time

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "SELECT payload,version,state,updated_time,error_code,error_details FROM xpc_group_config WHERE cpe_mac=? AND group_id=?"
	if err := c.Query(stmt, cpeMac, groupId).Scan(&payload, &version, &state, &updatedTime, &errorCode, &errorDetails); err != nil {
		return nil, common.NewError(err)
	}

	if len(payload) == 0 {
		return nil, common.NewError(gocql.ErrNotFound)
	}

	if c.IsEncryptedGroup(groupId) {
		payload, err = c.DecryptBytes(payload)
		if err != nil {
			return nil, common.NewError(err)
		}
	}
	ts := int(updatedTime.UnixNano() / 1000000)
	subdoc := common.NewSubDocument(payload, &version, &state, &ts, &errorCode, &errorDetails)
	return subdoc, nil
}

func (c *CassandraClient) SetSubDocument(cpeMac string, groupId string, subdoc *common.SubDocument, vargs ...interface{}) error {
	var oldState int
	metricsAgent := "default"
	for _, varg := range vargs {
		switch ty := varg.(type) {
		case int:
			oldState = ty
		case string:
			if len(ty) > 0 {
				metricsAgent = ty
			}
		}
	}
	var newStatePtr *int

	// build the statement and avoid unnecessary fields/columns
	columns := []string{"cpe_mac", "group_id"}
	values := []interface{}{cpeMac, groupId}
	if subdoc.Payload() != nil && len(subdoc.Payload()) > 0 {
		columns = append(columns, "payload")
		// TODO evel if it is necessary use a list of groupIds that need encryption
		if c.IsEncryptedGroup(groupId) {
			encbytes, err := c.EncryptBytes(subdoc.Payload())
			if err != nil {
				return common.NewError(err)
			}
			values = append(values, encbytes)
		} else {
			values = append(values, subdoc.Payload())
		}
	}
	if subdoc.Version() != nil {
		columns = append(columns, "version")
		values = append(values, subdoc.Version())
	}
	if subdoc.State() != nil {
		columns = append(columns, "state")
		values = append(values, subdoc.State())
		newStatePtr = subdoc.State()
	}
	if subdoc.UpdatedTime() != nil {
		columns = append(columns, "updated_time")
		utime := int64(*subdoc.UpdatedTime())
		if utime < 0 {
			err := fmt.Errorf("invalid updated_time: utime=%v, *subdoc.UpdatedTime()=%v", utime, *subdoc.UpdatedTime())
			return common.NewError(err)
		}
		values = append(values, &utime)
	}
	if subdoc.ErrorCode() != nil {
		columns = append(columns, "error_code")
		values = append(values, subdoc.ErrorCode())
	}
	if subdoc.ErrorDetails() != nil {
		columns = append(columns, "error_details")
		values = append(values, subdoc.ErrorDetails())
	}
	stmt := fmt.Sprintf("INSERT INTO xpc_group_config(%v) VALUES(%v)", db.GetColumnsStr(columns), db.GetValuesStr(len(columns)))

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	if err := c.Query(stmt, values...).Exec(); err != nil {
		return common.NewError(err)
	}

	// update state metrics
	if c.IsMetricsEnabled() {
		if newStatePtr != nil {
			c.UpdateStateMetrics(oldState, *newStatePtr, groupId, metricsAgent)
		}
	}
	return nil
}

func (c *CassandraClient) DeleteSubDocument(cpeMac string, groupId string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "DELETE FROM xpc_group_config WHERE cpe_mac=? AND group_id=?"
	if err := c.Query(stmt, cpeMac, groupId).Exec(); err != nil {
		return common.NewError(err)
	}
	return nil
}

func (c *CassandraClient) DeleteDocument(cpeMac string) error {
	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "DELETE FROM xpc_group_config WHERE cpe_mac=?"
	if err := c.Query(stmt, cpeMac).Exec(); err != nil {
		return common.NewError(err)
	}

	return nil
}

func (c *CassandraClient) GetDocument(cpeMac string) (*common.Document, error) {
	doc := common.NewDocument(nil)

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "SELECT group_id,payload,version,state,updated_time,error_code,error_details FROM xpc_group_config WHERE cpe_mac=?"
	iter := c.Query(stmt, cpeMac).Iter()

	for {
		var err error
		var payload []byte
		var groupId, version, errorDetails string
		var state, errorCode int
		var updatedTime time.Time

		if !iter.Scan(&groupId, &payload, &version, &state, &updatedTime, &errorCode, &errorDetails) {
			break
		}

		if len(payload) == 0 {
			continue
		}

		if c.IsEncryptedGroup(groupId) {
			payload, err = c.DecryptBytes(payload)
			if err != nil {
				return nil, common.NewError(err)
			}
		}

		ts := int(updatedTime.UnixNano() / 1000000)

		subdoc := common.NewSubDocument(payload, &version, &state, &ts, &errorCode, &errorDetails)
		doc.SetSubDocument(groupId, subdoc)
	}

	if doc.Length() == 0 {
		return doc, common.NewError(gocql.ErrNotFound)
	}
	return doc, nil
}
