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
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestMocaSubDocument(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	subdocId := "moca"

	// prepare the source data
	srcBytes := common.RandomBytes(16, 116)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	srcState := common.PendingDownload

	// verify empty before start
	fields := log.Fields{}
	var err error
	_, err = tdbclient.GetSubDocument(cpeMac, subdocId)
	assert.Assert(t, tdbclient.IsDbNotFound(err))

	// write into db
	srcSubdoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	err = tdbclient.SetSubDocument(cpeMac, subdocId, srcSubdoc, fields)
	assert.NilError(t, err)

	fetchedSubdoc, err := tdbclient.GetSubDocument(cpeMac, subdocId)
	assert.NilError(t, err)

	ok, err := srcSubdoc.Equals(fetchedSubdoc)
	assert.NilError(t, err)
	assert.Assert(t, ok)
}

func TestPrivatessidSubDocument(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "privatessid"

	srcBytes := common.RandomBytes(16, 116)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	srcState := common.PendingDownload

	// write into db
	srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)

	fields := log.Fields{}
	var err error
	err = tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
	assert.NilError(t, err)

	fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)

	ok, err := srcDoc.Equals(fetchedDoc)
	assert.NilError(t, err)
	assert.Assert(t, ok)
}

func TestMultiSubDocuments(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// prepare the source data
	queryStr := "privatessid,homessid,moca"
	mparts := util.GetMockMultiparts(queryStr)
	assert.Equal(t, len(mparts), 3)

	srcmap := make(map[string]common.SubDocument)

	fields := log.Fields{}
	for _, mpart := range mparts {
		groupId := mpart.Name
		srcBytes := mpart.Bytes
		srcVersion := util.GetMurmur3Hash(srcBytes)
		srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
		srcState := common.PendingDownload

		// write into db
		// enforce "params" to be non-empty
		srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
		srcmap[groupId] = *srcDoc

		err := tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
		assert.NilError(t, err)

		fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
		assert.NilError(t, err)

		ok, err := srcDoc.Equals(fetchedDoc)
		assert.NilError(t, err)
		assert.Assert(t, ok)
	}

	doc, err := tdbclient.GetDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, len(srcmap), doc.Length())

	for k, v := range srcmap {
		dv := doc.SubDocument(k)
		assert.Assert(t, dv != nil)
		ok, err := v.Equals(dv)
		assert.NilError(t, err)
		assert.Assert(t, ok)
	}

	// ==== delete a document ====
	err = tdbclient.DeleteSubDocument(cpeMac, "moca")
	assert.NilError(t, err)

	doc, err = tdbclient.GetDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, len(srcmap)-1, doc.Length())

	subdoc := doc.SubDocument("moca")
	assert.Assert(t, subdoc == nil)

	err = tdbclient.DeleteDocument(cpeMac)
	assert.NilError(t, err)

	_, err = tdbclient.GetDocument(cpeMac)
	assert.Assert(t, tdbclient.IsDbNotFound(err))
}

func TestBlockedSubdocIds(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	blockedSubdocIds := []string{"portforwarding", "macbinding"}
	tdbclient.SetBlockedSubdocIds(blockedSubdocIds)

	// prepare the source data
	queryStr := "privatessid,homessid,moca,portforwarding,macbinding"
	mparts := util.GetMockMultiparts(queryStr)
	assert.Equal(t, len(mparts), 5)

	srcmap := make(map[string]common.SubDocument)

	fields := log.Fields{}
	for _, mpart := range mparts {
		groupId := mpart.Name
		srcBytes := mpart.Bytes
		srcVersion := util.GetMurmur3Hash(srcBytes)
		srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
		srcState := common.PendingDownload

		// write into db
		// enforce "params" to be non-empty
		srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
		srcmap[groupId] = *srcDoc

		err := tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
		assert.NilError(t, err)

		fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
		assert.NilError(t, err)

		ok, err := srcDoc.Equals(fetchedDoc)
		assert.NilError(t, err)
		assert.Assert(t, ok)
	}
	// add version1 and bitmap1
	version1 := "indigo violet"
	err := tdbclient.SetRootDocumentVersion(cpeMac, version1)
	assert.NilError(t, err)

	bitmap1 := 32479
	err = tdbclient.SetRootDocumentBitmap(cpeMac, bitmap1)
	assert.NilError(t, err)

	// ==== read to verify ====

	rHeader := make(http.Header)
	rHeader.Set(common.HeaderDeviceId, cpeMac)
	rdkSupportedDocsHeaderStr := "16777247,33554435,50331649,67108865,83886081,100663297,117440513,134217729"

	rHeader.Set(common.HeaderSupportedDocs, rdkSupportedDocsHeaderStr)

	document, _, _, _, _, _, err := db.BuildGetDocument(tdbclient, rHeader, common.RouteHttp, fields)
	assert.NilError(t, err)
	assert.Assert(t, document.Length() == 3)
	versionMap := document.VersionMap()
	_, ok := versionMap["portforwarding"]
	assert.Assert(t, !ok)
	_, ok = versionMap["macbinding"]
	assert.Assert(t, !ok)

	tdbclient.SetBlockedSubdocIds([]string{})
	assert.Equal(t, len(tdbclient.BlockedSubdocIds()), 0)
}

func TestExpirySubDocument(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// verify empty before start
	fields := log.Fields{}
	var err error
	_, err = tdbclient.GetDocument(cpeMac)
	assert.Assert(t, tdbclient.IsDbNotFound(err))

	// prepare some subdocs
	subdocIds := []string{"privatessid", "lan", "wan"}
	for _, subdocId := range subdocIds {
		srcBytes := common.RandomBytes(100, 150)
		srcVersion := util.GetMurmur3Hash(srcBytes)
		srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
		srcState := common.PendingDownload
		srcSubdoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
		err = tdbclient.SetSubDocument(cpeMac, subdocId, srcSubdoc, fields)
		assert.NilError(t, err)
	}

	// read the document
	doc, err := tdbclient.GetDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, doc.Length() == len(subdocIds))

	// add an expiry-type but not-yet-expired subdoc
	srcBytes := common.RandomBytes(100, 150)
	now := time.Now()
	nowTs := int(now.UnixNano() / 1000000)
	futureT := now.AddDate(0, 0, 2)
	futureTs := int(futureT.UnixNano() / 1000000)
	srcUpdatedTime := nowTs
	srcVersion := strconv.Itoa(nowTs)
	srcState := common.PendingDownload
	srcSubdoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	srcSubdoc.SetExpiry(&futureTs)
	err = tdbclient.SetSubDocument(cpeMac, "mesh", srcSubdoc, fields)
	assert.NilError(t, err)

	// read the document
	doc, err = tdbclient.GetDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, doc.Length() == len(subdocIds)+1)

	// set an expired subdoc
	srcBytes = common.RandomBytes(100, 150)
	past := now.Add(time.Duration(-1) * time.Hour)
	pastTs := int(past.UnixNano() / 1000000)
	srcVersion = strconv.Itoa(nowTs)

	srcUpdatedTime = nowTs
	srcState = common.PendingDownload
	srcSubdoc = common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	srcSubdoc.SetExpiry(&pastTs)
	err = tdbclient.SetSubDocument(cpeMac, "gwrestore", srcSubdoc, fields)
	assert.NilError(t, err)

	// read the document
	doc, err = tdbclient.GetDocument(cpeMac)
	assert.NilError(t, err)
	assert.Assert(t, doc.Length() == len(subdocIds)+1)
}

func TestGetSubDocumentWithReference(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	subdocId := "lan"
	refId := util.GetMurmur3Hash([]byte(cpeMac + subdocId))

	// Step 1: Create a reference subdocument with actual payload
	actualPayload := common.RandomBytes(100, 200)
	actualVersion := util.GetMurmur3Hash(actualPayload)
	refSubdoc := common.NewRefSubDocument(actualPayload, &actualVersion)

	err := tdbclient.SetRefSubDocument(refId, refSubdoc)
	assert.NilError(t, err)

	// Step 2: Create a subdocument with reference payload (4 zero bytes + refId)
	referencePayload := append(make([]byte, 4), []byte(refId)...)
	refVersion := util.GetMurmur3Hash(referencePayload)
	refState := common.InDeployment
	refUpdatedTime := int(time.Now().UnixMilli())

	subdocWithRef := common.NewSubDocument(referencePayload, &refVersion, &refState, &refUpdatedTime, nil, nil)
	fields := log.Fields{}
	err = tdbclient.SetSubDocument(cpeMac, subdocId, subdocWithRef, fields)
	assert.NilError(t, err)

	// Step 3: Call GetSubDocument and verify it returns the actual payload, not the reference
	fetchedSubdoc, err := tdbclient.GetSubDocument(cpeMac, subdocId)
	assert.NilError(t, err)
	assert.Assert(t, fetchedSubdoc != nil)

	// Verify the payload is the actual payload from refsubdocument, not the reference
	assert.DeepEqual(t, fetchedSubdoc.Payload(), actualPayload)

	// Verify other fields remain unchanged
	assert.Equal(t, *fetchedSubdoc.Version(), refVersion)
	assert.Equal(t, *fetchedSubdoc.State(), refState)
	assert.Equal(t, *fetchedSubdoc.UpdatedTime(), refUpdatedTime)

	// Cleanup
	err = tdbclient.DeleteSubDocument(cpeMac, subdocId)
	assert.NilError(t, err)
	err = tdbclient.DeleteRefSubDocument(refId)
	assert.NilError(t, err)
}

func TestGetSubDocumentWithMissingReference(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	subdocId := "wan"
	refId := util.GetMurmur3Hash([]byte(cpeMac + subdocId + "nonexistent"))

	// Create a subdocument with reference payload pointing to non-existent refsubdocument
	referencePayload := append(make([]byte, 4), []byte(refId)...)
	refVersion := util.GetMurmur3Hash(referencePayload)
	refState := common.InDeployment
	refUpdatedTime := int(time.Now().UnixMilli())

	subdocWithRef := common.NewSubDocument(referencePayload, &refVersion, &refState, &refUpdatedTime, nil, nil)
	fields := log.Fields{}
	err := tdbclient.SetSubDocument(cpeMac, subdocId, subdocWithRef, fields)
	assert.NilError(t, err)

	// Call GetSubDocument - should return the reference payload since refsubdocument doesn't exist
	fetchedSubdoc, err := tdbclient.GetSubDocument(cpeMac, subdocId)
	assert.NilError(t, err)
	assert.Assert(t, fetchedSubdoc != nil)

	// Verify the payload is the reference payload (since refsubdocument was not found)
	assert.DeepEqual(t, fetchedSubdoc.Payload(), referencePayload)

	// Cleanup
	err = tdbclient.DeleteSubDocument(cpeMac, subdocId)
	assert.NilError(t, err)
}

func TestGetSubDocumentWithoutReference(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	subdocId := "mesh"

	// Create a regular subdocument without any reference
	regularPayload := common.RandomBytes(100, 200)
	regularVersion := util.GetMurmur3Hash(regularPayload)
	regularState := common.Deployed
	regularUpdatedTime := int(time.Now().UnixMilli())

	subdoc := common.NewSubDocument(regularPayload, &regularVersion, &regularState, &regularUpdatedTime, nil, nil)
	fields := log.Fields{}
	err := tdbclient.SetSubDocument(cpeMac, subdocId, subdoc, fields)
	assert.NilError(t, err)

	// Call GetSubDocument - should return the regular payload unchanged
	fetchedSubdoc, err := tdbclient.GetSubDocument(cpeMac, subdocId)
	assert.NilError(t, err)
	assert.Assert(t, fetchedSubdoc != nil)

	// Verify the payload is unchanged
	assert.DeepEqual(t, fetchedSubdoc.Payload(), regularPayload)
	assert.Equal(t, *fetchedSubdoc.Version(), regularVersion)
	assert.Equal(t, *fetchedSubdoc.State(), regularState)
	assert.Equal(t, *fetchedSubdoc.UpdatedTime(), regularUpdatedTime)

	// Cleanup
	err = tdbclient.DeleteSubDocument(cpeMac, subdocId)
	assert.NilError(t, err)
}
