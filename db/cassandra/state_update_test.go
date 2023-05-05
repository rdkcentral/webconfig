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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestStateUpdate1(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// setup a doct
	groupId := "privatessid"
	srcBytes := util.RandomBytes(100, 150)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	srcState := common.PendingDownload

	srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
	assert.NilError(t, err)

	fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)

	err = srcDoc.Equals(fetchedDoc)
	assert.NilError(t, err)

	// update to state failure
	template1 := `{"application_status": "failure", "error_code": 204, "error_details": "failed_retrying:Error unsupported namespace", "device_id": "mac:%v", "namespace": "privatessid", "version": "2023-05-05 07:42:22.515324", "transaction_uuid": "becd74ee-2c17-4abe-aa60-332a218c91aa"}`
	bbytes := []byte(fmt.Sprintf(template1, cpeMac))
	var m common.EventMessage
	err = json.Unmarshal(bbytes, &m)
	assert.NilError(t, err)
	err = db.UpdateDocumentState(tdbclient, cpeMac, &m, fields)
	assert.NilError(t, err)
	subdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Failure)
	assert.Equal(t, *subdoc.ErrorCode(), 204)
	assert.Equal(t, *subdoc.ErrorDetails(), "failed_retrying:Error unsupported namespace")

	// update to state success
	template2 := `{"application_status": "success", "device_id": "mac:%v", "namespace": "privatessid", "version": "2023-05-05 07:42:11.959437", "transaction_uuid": "0dd08490-7ab6-4080-b153-78ecef4412f6"}`
	bbytes = []byte(fmt.Sprintf(template2, cpeMac))
	m = common.EventMessage{}
	err = json.Unmarshal(bbytes, &m)
	assert.NilError(t, err)
	err = db.UpdateDocumentState(tdbclient, cpeMac, &m, fields)
	assert.NilError(t, err)
	subdoc, err = tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Deployed)
	assert.Equal(t, *subdoc.ErrorCode(), 0)
	assert.Equal(t, *subdoc.ErrorDetails(), "")
}

func TestStateUpdate2(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// setup a doct
	groupId := "privatessid"
	srcBytes := util.RandomBytes(100, 150)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	srcState := common.PendingDownload

	srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
	assert.NilError(t, err)

	fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)

	err = srcDoc.Equals(fetchedDoc)
	assert.NilError(t, err)

	// update to state failure
	template1 := `{"application_status": "failure", "error_code": 204, "error_details": "failed_retrying:Error unsupported namespace", "device_id": "mac:%v", "namespace": "privatessid", "version": "2023-05-05 07:42:22.515324", "transaction_uuid": "becd74ee-2c17-4abe-aa60-332a218c91aa"}`
	bbytes := []byte(fmt.Sprintf(template1, cpeMac))
	var m common.EventMessage
	err = json.Unmarshal(bbytes, &m)
	assert.NilError(t, err)
	err = db.UpdateDocumentState(tdbclient, cpeMac, &m, fields)
	assert.NilError(t, err)
	subdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Failure)
	assert.Equal(t, *subdoc.ErrorCode(), 204)
	assert.Equal(t, *subdoc.ErrorDetails(), "failed_retrying:Error unsupported namespace")

	// update to state success by http 304
	template2 := `{"device_id": "mac:%v", "http_status_code": 304, "transaction_uuid": "352b85d0-d479-4704-8f9a-bef78b1e7fbf", "version": "2023-05-05 07:42:50.395876"}`
	bbytes = []byte(fmt.Sprintf(template2, cpeMac))
	m = common.EventMessage{}
	err = json.Unmarshal(bbytes, &m)
	assert.NilError(t, err)
	err = db.UpdateDocumentState(tdbclient, cpeMac, &m, fields)
	assert.NilError(t, err)
	subdoc, err = tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Deployed)
	assert.Equal(t, *subdoc.ErrorCode(), 0)
	assert.Equal(t, *subdoc.ErrorDetails(), "")
}
