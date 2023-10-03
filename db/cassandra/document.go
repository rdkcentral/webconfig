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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
)

// NOTE this
func (c *CassandraClient) GetSubDocument(cpeMac string, groupId string) (*common.SubDocument, error) {
	var err error
	var payload []byte
	var version, errorDetails string
	var state, errorCode int
	var updatedTime, expiry time.Time
	var updatedTimeTsPtr, expiryTsPtr *int

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "SELECT payload,version,state,updated_time,error_code,error_details,expiry FROM xpc_group_config WHERE cpe_mac=? AND group_id=?"
	if err := c.Query(stmt, cpeMac, groupId).Scan(&payload, &version, &state, &updatedTime, &errorCode, &errorDetails, &expiry); err != nil {
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

	if x := int(updatedTime.UnixNano() / 1000000); x > 0 {
		updatedTimeTsPtr = &x
	}

	subdoc := common.NewSubDocument(payload, &version, &state, updatedTimeTsPtr, &errorCode, &errorDetails)
	if x := int(expiry.UnixNano() / 1000000); x > 0 {
		expiryTsPtr = &x
	}
	if expiryTsPtr != nil {
		subdoc.SetExpiry(expiryTsPtr)
	}

	return subdoc, nil
}

func (c *CassandraClient) SetSubDocument(cpeMac string, groupId string, subdoc *common.SubDocument, vargs ...interface{}) (fnerr error) {
	var oldState int
	var fields log.Fields
	var labels prometheus.Labels
	for _, varg := range vargs {
		switch ty := varg.(type) {
		case int:
			oldState = ty
		case log.Fields:
			fields = ty
		case prometheus.Labels:
			labels = ty
			// should include only "model", "fwversion" and "client"
		}
	}
	var newStatePtr *int
	var stmt string
	columnMap := util.Dict{
		"cpe_mac":  cpeMac,
		"group_id": groupId,
	}
	defer func() {
		var tfields log.Fields
		if fields == nil {
			tfields = make(log.Fields)
		} else {
			tfields = common.FilterLogFields(fields)
		}
		tfields["logger"] = "xdb"
		columnMap["stmt"] = stmt
		tfields["query"] = columnMap
		tfields["func_err"] = fnerr
		log.WithFields(tfields).Debug("SetSubDocument()")
	}()

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
			columnMap["payload_len"] = len(encbytes)
		} else {
			values = append(values, subdoc.Payload())
			columnMap["payload_len"] = len(subdoc.Payload())
		}
	}
	if subdoc.Version() != nil {
		columns = append(columns, "version")
		values = append(values, subdoc.Version())
		columnMap["version"] = subdoc.Version()
	}
	if subdoc.State() != nil {
		columns = append(columns, "state")
		values = append(values, subdoc.State())
		newStatePtr = subdoc.State()
		columnMap["state"] = subdoc.State()
	}
	if subdoc.UpdatedTime() != nil {
		columns = append(columns, "updated_time")
		utime := int64(*subdoc.UpdatedTime())
		if utime < 0 {
			err := fmt.Errorf("invalid updated_time: utime=%v, *subdoc.UpdatedTime()=%v", utime, *subdoc.UpdatedTime())
			return common.NewError(err)
		}
		values = append(values, &utime)
		columnMap["updated_time"] = utime
	}
	if subdoc.ErrorCode() != nil {
		columns = append(columns, "error_code")
		values = append(values, subdoc.ErrorCode())
		columnMap["error_code"] = subdoc.ErrorCode()
	}
	if subdoc.ErrorDetails() != nil {
		columns = append(columns, "error_details")
		values = append(values, subdoc.ErrorDetails())
		columnMap["error_details"] = subdoc.ErrorDetails()
	}
	if subdoc.Expiry() != nil {
		columns = append(columns, "expiry")
		utime := int64(*subdoc.Expiry())
		if utime < 0 {
			err := fmt.Errorf("invalid expiry: utime=%v, *subdoc.Expiry()=%v", utime, *subdoc.Expiry())
			return common.NewError(err)
		}
		values = append(values, &utime)
		columnMap["expiry"] = utime
	}
	stmt = fmt.Sprintf("INSERT INTO xpc_group_config(%v) VALUES(%v)", db.GetColumnsStr(columns), db.GetValuesStr(len(columns)))

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	if err := c.Query(stmt, values...).Exec(); err != nil {
		return common.NewError(err)
	}

	// update state metrics
	if c.IsMetricsEnabled() {
		if newStatePtr != nil {
			labels["feature"] = groupId
			c.UpdateStateMetrics(oldState, *newStatePtr, labels, cpeMac, fields)
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

func (c *CassandraClient) GetDocument(cpeMac string, xargs ...interface{}) (fndoc *common.Document, fnerr error) {
	var includeExpiry bool
	var fields log.Fields
	if len(xargs) > 0 {
		for _, v := range xargs {
			switch t := v.(type) {
			case bool:
				includeExpiry = t
			case log.Fields:
				fields = t
			}
		}
	}

	doc := common.NewDocument(nil)

	c.concurrentQueries <- true
	defer func() { <-c.concurrentQueries }()

	stmt := "SELECT group_id,payload,version,state,updated_time,error_code,error_details,expiry FROM xpc_group_config WHERE cpe_mac=?"
	iter := c.Query(stmt, cpeMac).Iter()
	rmap := make(util.Dict)
	defer func() {
		var tfields log.Fields
		if fields == nil {
			tfields = make(log.Fields)
		} else {
			tfields = common.FilterLogFields(fields)
		}
		tfields["logger"] = "xdb"
		tfields["query_stmt"] = stmt
		tfields["query_result"] = rmap
		tfields["func_err"] = fnerr
		delete(tfields, "document")
		delete(tfields, "header")
		delete(fields, "src_caller")
		delete(fields, "document")
		log.WithFields(tfields).Debug("GetDocument()")
	}()

	now := time.Now()
	for {
		var err error
		var payload []byte
		var groupId, version, errorDetails string
		var state, errorCode int
		var updatedTime, expiry time.Time
		var updatedTimeTsPtr *int

		if !iter.Scan(&groupId, &payload, &version, &state, &updatedTime, &errorCode, &errorDetails, &expiry) {
			break
		}

		// build the logging obj
		row := util.Dict{
			"version":     version,
			"state":       state,
			"payload_len": len(payload),
		}
		if !updatedTime.IsZero() {
			row["updated_time"] = updatedTime.Format(common.LoggingTimeFormat)
		}
		if !expiry.IsZero() {
			row["expiry"] = expiry.Format(common.LoggingTimeFormat)
		}
		row["payload_len"] = len(payload)
		rmap[groupId] = row

		if len(payload) == 0 {
			continue
		}

		if c.IsEncryptedGroup(groupId) {
			payload, err = c.DecryptBytes(payload)
			if err != nil {
				return nil, common.NewError(err)
			}
		}

		if x := int(updatedTime.UnixNano() / 1000000); x > 0 {
			updatedTimeTsPtr = &x
		}

		subdoc := common.NewSubDocument(payload, &version, &state, updatedTimeTsPtr, &errorCode, &errorDetails)
		// REMINDER, need this operation to detect if the "expiry" column is null/empty
		if !expiry.IsZero() {
			if x := int(expiry.UnixNano() / 1000000); x > 0 {
				// eval subdocs with expiry
				if !includeExpiry {
					if expiry.Before(now) {
						continue
					}
				}
				subdoc.SetExpiry(&x)
			}
		}

		doc.SetSubDocument(groupId, subdoc)
	}

	if fields != nil {
		fields["document"] = rmap
	}

	if doc.Length() == 0 {
		return doc, common.NewError(gocql.ErrNotFound)
	}
	return doc, nil
}
