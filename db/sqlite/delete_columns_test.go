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
	"testing"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestDeleteSubDocumentColumnsExpiry(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "telemetry"

	// Create subdocument with expiry
	srcBytes := common.RandomBytes(16, 116)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	srcState := common.Deployed
	futureExpiry := int(time.Now().Add(24*time.Hour).UnixNano() / 1000000)

	fields := log.Fields{}

	// Create subdocument with expiry set
	srcSubdoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	srcSubdoc.SetExpiry(&futureExpiry)

	// Write to database
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcSubdoc, fields)
	assert.NilError(t, err)

	// Verify expiry is set
	fetchedSubdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Assert(t, fetchedSubdoc.Expiry() != nil)
	assert.Equal(t, *fetchedSubdoc.Expiry(), futureExpiry)

	// Delete the expiry column
	err = tdbclient.DeleteSubDocumentColumns(cpeMac, groupId, "expiry")
	assert.NilError(t, err)

	// Verify expiry is now nil
	fetchedSubdoc2, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Assert(t, fetchedSubdoc2.Expiry() == nil)

	// Verify other fields are unchanged
	assert.Equal(t, *fetchedSubdoc2.Version(), srcVersion)
	assert.Equal(t, *fetchedSubdoc2.State(), srcState)
	assert.Equal(t, len(fetchedSubdoc2.Payload()), len(srcBytes))
}

func TestDeleteSubDocumentColumnsMultiple(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "mesh"

	// Create subdocument with expiry
	srcBytes := common.RandomBytes(16, 116)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	srcState := common.InDeployment
	futureExpiry := int(time.Now().Add(24*time.Hour).UnixNano() / 1000000)

	fields := log.Fields{}

	// Create subdocument with expiry set
	srcSubdoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	srcSubdoc.SetExpiry(&futureExpiry)

	// Write to database
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcSubdoc, fields)
	assert.NilError(t, err)

	// Verify expiry is set
	fetchedSubdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Assert(t, fetchedSubdoc.Expiry() != nil)

	// Delete expiry column
	err = tdbclient.DeleteSubDocumentColumns(cpeMac, groupId, "expiry")
	assert.NilError(t, err)

	// Verify expiry is now nil
	fetchedSubdoc2, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Assert(t, fetchedSubdoc2.Expiry() == nil)

	// Verify other fields are unchanged
	assert.Equal(t, *fetchedSubdoc2.Version(), srcVersion)
	assert.Equal(t, *fetchedSubdoc2.State(), srcState)
	assert.Equal(t, len(fetchedSubdoc2.Payload()), len(srcBytes))
}

func TestDeleteSubDocumentColumnsEmptyList(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "test"

	// Create a simple subdocument
	srcBytes := common.RandomBytes(16, 116)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcState := common.PendingDownload
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)

	fields := log.Fields{}

	srcSubdoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcSubdoc, fields)
	assert.NilError(t, err)

	// Call with empty column list should be no-op
	err = tdbclient.DeleteSubDocumentColumns(cpeMac, groupId)
	assert.NilError(t, err)

	// Verify subdocument is unchanged
	fetchedSubdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *fetchedSubdoc.Version(), srcVersion)
	assert.Equal(t, *fetchedSubdoc.State(), srcState)
}

func TestDeleteSubDocumentColumnsErrorFields(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "telemetry"

	// Create subdocument with error fields and expiry
	srcBytes := common.RandomBytes(16, 116)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	srcState := common.Failure
	errorCode := 204
	errorDetails := "failed_retrying:Error unsupported namespace"
	futureExpiry := int(time.Now().Add(24*time.Hour).UnixNano() / 1000000)

	fields := log.Fields{}

	// Create subdocument with error fields and expiry set
	srcSubdoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, &errorCode, &errorDetails)
	srcSubdoc.SetExpiry(&futureExpiry)

	// Write to database
	err := tdbclient.SetSubDocument(cpeMac, groupId, srcSubdoc, fields)
	assert.NilError(t, err)

	// Verify all fields are set
	fetchedSubdoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Assert(t, fetchedSubdoc.Expiry() != nil)
	assert.Equal(t, *fetchedSubdoc.Expiry(), futureExpiry)
	assert.Assert(t, fetchedSubdoc.ErrorCode() != nil)
	assert.Equal(t, *fetchedSubdoc.ErrorCode(), errorCode)
	assert.Assert(t, fetchedSubdoc.ErrorDetails() != nil)
	assert.Equal(t, *fetchedSubdoc.ErrorDetails(), errorDetails)

	// Delete expiry and error fields
	err = tdbclient.DeleteSubDocumentColumns(cpeMac, groupId, "expiry", "error_code", "error_details")
	assert.NilError(t, err)

	// Verify deleted columns are now nil
	fetchedSubdoc2, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Assert(t, fetchedSubdoc2.Expiry() == nil)
	assert.Assert(t, fetchedSubdoc2.ErrorCode() == nil)
	assert.Assert(t, fetchedSubdoc2.ErrorDetails() == nil)

	// Verify other fields are unchanged
	assert.Equal(t, *fetchedSubdoc2.Version(), srcVersion)
	assert.Equal(t, *fetchedSubdoc2.State(), srcState)
	assert.Equal(t, len(fetchedSubdoc2.Payload()), len(srcBytes))
}
