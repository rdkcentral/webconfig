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
	"crypto/rand"
	"net/http"
	"strconv"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestMocaSubDocument(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	subdocId := "moca"

	// prepare the source data
	slen := util.RandomInt(100) + 16
	srcBytes := make([]byte, slen)
	rand.Read(srcBytes)
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

	assert.Assert(t, srcSubdoc.Equals(fetchedSubdoc))
}

func TestPrivatessidSubDocument(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "privatessid"

	slen := util.RandomInt(100) + 16
	srcBytes := make([]byte, slen)
	rand.Read(srcBytes)
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

	assert.Assert(t, srcDoc.Equals(fetchedDoc))
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

		err = srcDoc.Equals(fetchedDoc)
		assert.NilError(t, err)
	}

	doc, err := tdbclient.GetDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, len(srcmap), doc.Length())

	for k, v := range srcmap {
		dv := doc.SubDocument(k)
		assert.Assert(t, dv != nil)
		assert.Assert(t, v.Equals(dv))
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

		err = srcDoc.Equals(fetchedDoc)
		assert.NilError(t, err)
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

	document, _, _, _, _, err := db.BuildGetDocument(tdbclient, rHeader, common.RouteHttp, fields)
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
		srcBytes := util.RandomBytes(100, 150)
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
	srcBytes := util.RandomBytes(100, 150)
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
	srcBytes = util.RandomBytes(100, 150)
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
