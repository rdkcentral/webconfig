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

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestTelemetryStateUpdateWithTTL(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// setup a telemetry subdoc
	groupId := "telemetry"
	srcBytes := common.RandomBytes(100, 150)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixMilli())
	srcState := common.PendingDownload

	srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
	assert.NilError(t, err)

	fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)

	ok, err := srcDoc.Equals(fetchedDoc)
	assert.NilError(t, err)
	assert.Assert(t, ok)

	// Update to success with TTL set to 7 days
	template := `{"application_status": "success", "device_id": "mac:%v", "namespace": "telemetry", "version": "%v", "transaction_uuid": "0dd08490-7ab6-4080-b153-78ecef4412f6"}`
	bbytes := []byte(fmt.Sprintf(template, cpeMac, srcVersion))
	var m common.EventMessage
	err = json.Unmarshal(bbytes, &m)
	assert.NilError(t, err)

	// Enable supplementary precook and set TTL to 7 days on database client
	supplementaryPrecookStateTTLDays := 7
	tdbclient.SetSupplementaryPrecookEnabled(true)
	tdbclient.SetSupplementaryPrecookStateTTLDays(supplementaryPrecookStateTTLDays)
	updatedSubdocIds, err := db.UpdateDocumentState(tdbclient, cpeMac, &m, fields)
	assert.NilError(t, err)
	assert.Assert(t, len(updatedSubdocIds) == 0)

	subdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Deployed)
	assert.Equal(t, *subdoc.ErrorCode(), 0)
	assert.Equal(t, *subdoc.ErrorDetails(), "")

	// Verify that expiry was set
	assert.Assert(t, subdoc.Expiry() != nil)
	expiryTime := *subdoc.Expiry()
	currentTime := int(time.Now().UnixMilli())
	expectedExpiry := int(time.Now().Add(time.Duration(supplementaryPrecookStateTTLDays) * 24 * time.Hour).UnixMilli())

	// Allow some margin for execution time (10 seconds)
	margin := int(10 * time.Second / time.Millisecond)
	assert.Assert(t, expiryTime >= currentTime)
	assert.Assert(t, expiryTime <= expectedExpiry+margin)
}

func TestTelemetryStateUpdateWithoutTTL(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// setup a telemetry subdoc
	groupId := "telemetry"
	srcBytes := common.RandomBytes(100, 150)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixMilli())
	srcState := common.PendingDownload

	srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
	assert.NilError(t, err)

	fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)

	ok, err := srcDoc.Equals(fetchedDoc)
	assert.NilError(t, err)
	assert.Assert(t, ok)

	// Update to success with TTL set to 0 (no TTL)
	template := `{"application_status": "success", "device_id": "mac:%v", "namespace": "telemetry", "version": "%v", "transaction_uuid": "0dd08490-7ab6-4080-b153-78ecef4412f6"}`
	bbytes := []byte(fmt.Sprintf(template, cpeMac, srcVersion))
	var m common.EventMessage
	err = json.Unmarshal(bbytes, &m)
	assert.NilError(t, err)

	// Enable supplementary precook and set TTL to 0 (no expiry)
	supplementaryPrecookStateTTLDays := 0
	tdbclient.SetSupplementaryPrecookEnabled(true)
	tdbclient.SetSupplementaryPrecookStateTTLDays(supplementaryPrecookStateTTLDays)
	updatedSubdocIds, err := db.UpdateDocumentState(tdbclient, cpeMac, &m, fields)
	assert.NilError(t, err)
	assert.Assert(t, len(updatedSubdocIds) == 0)

	subdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Deployed)
	assert.Equal(t, *subdoc.ErrorCode(), 0)
	assert.Equal(t, *subdoc.ErrorDetails(), "")

	// Verify that expiry was NOT set
	assert.Assert(t, subdoc.Expiry() == nil)
}

func TestNonTelemetryStateUpdateNoTTL(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// setup a non-telemetry subdoc (e.g., privatessid)
	groupId := "privatessid"
	srcBytes := common.RandomBytes(100, 150)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixMilli())
	srcState := common.PendingDownload

	srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
	assert.NilError(t, err)

	fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)

	ok, err := srcDoc.Equals(fetchedDoc)
	assert.NilError(t, err)
	assert.Assert(t, ok)

	// Update to success with TTL configured but for non-telemetry subdoc
	template := `{"application_status": "success", "device_id": "mac:%v", "namespace": "privatessid", "version": "%v", "transaction_uuid": "0dd08490-7ab6-4080-b153-78ecef4412f6"}`
	bbytes := []byte(fmt.Sprintf(template, cpeMac, srcVersion))
	var m common.EventMessage
	err = json.Unmarshal(bbytes, &m)
	assert.NilError(t, err)

	// Enable supplementary precook and set TTL to 7 days
	// but since it's not telemetry, TTL should NOT be set
	supplementaryPrecookStateTTLDays := 7
	tdbclient.SetSupplementaryPrecookEnabled(true)
	tdbclient.SetSupplementaryPrecookStateTTLDays(supplementaryPrecookStateTTLDays)
	updatedSubdocIds, err := db.UpdateDocumentState(tdbclient, cpeMac, &m, fields)
	assert.NilError(t, err)
	assert.Assert(t, len(updatedSubdocIds) == 0)

	subdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Deployed)
	assert.Equal(t, *subdoc.ErrorCode(), 0)
	assert.Equal(t, *subdoc.ErrorDetails(), "")

	// Verify that expiry was NOT set (since it's not telemetry)
	assert.Assert(t, subdoc.Expiry() == nil)
}

func TestTelemetryStateUpdateFailureNoTTL(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// setup a telemetry subdoc
	groupId := "telemetry"
	srcBytes := common.RandomBytes(100, 150)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixMilli())
	srcState := common.PendingDownload

	srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	fields := make(log.Fields)
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
	assert.NilError(t, err)

	fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)

	ok, err := srcDoc.Equals(fetchedDoc)
	assert.NilError(t, err)
	assert.Assert(t, ok)

	// Update to failure - TTL should NOT be set even with supplementaryPrecookStateTTLDays configured
	template := `{"application_status": "failure", "error_code": 204, "error_details": "failed_retrying:Error unsupported namespace", "device_id": "mac:%v", "namespace": "telemetry", "version": "%v", "transaction_uuid": "becd74ee-2c17-4abe-aa60-332a218c91aa"}`
	bbytes := []byte(fmt.Sprintf(template, cpeMac, srcVersion))
	var m common.EventMessage
	err = json.Unmarshal(bbytes, &m)
	assert.NilError(t, err)

	// Enable supplementary precook and set TTL to 7 days
	// But since state is failure, TTL should NOT be set
	supplementaryPrecookStateTTLDays := 7
	tdbclient.SetSupplementaryPrecookEnabled(true)
	tdbclient.SetSupplementaryPrecookStateTTLDays(supplementaryPrecookStateTTLDays)
	updatedSubdocIds, err := db.UpdateDocumentState(tdbclient, cpeMac, &m, fields)
	assert.NilError(t, err)
	assert.Assert(t, len(updatedSubdocIds) == 0)

	subdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *subdoc.State(), common.Failure)
	assert.Equal(t, *subdoc.ErrorCode(), 204)
	assert.Equal(t, *subdoc.ErrorDetails(), "failed_retrying:Error unsupported namespace")

	// Verify that expiry was NOT set (since state is failure, not success)
	assert.Assert(t, subdoc.Expiry() == nil)
}
