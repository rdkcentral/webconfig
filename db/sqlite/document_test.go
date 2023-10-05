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

func TestSubDocumentDb(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "privatessid"

	// verify starting empty
	fields := log.Fields{}
	_, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.Assert(t, tdbclient.IsDbNotFound(err))

	// ==== insert a doc ====
	srcBytes := []byte("hello world")
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	srcState := common.PendingDownload
	sourceDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
	err = tdbclient.SetSubDocument(cpeMac, groupId, sourceDoc, fields)
	assert.NilError(t, err)

	// read a SubDocument from db and verify identical
	targetSubDocument, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	err = sourceDoc.Equals(targetSubDocument)
	assert.NilError(t, err)

	// ==== update an existing doc with the same cpeMac and groupId ====
	srcVersion2 := "red white blue"
	sourceDoc2 := common.NewSubDocument(nil, &srcVersion2, nil, nil, nil, nil)
	err = tdbclient.SetSubDocument(cpeMac, groupId, sourceDoc2, fields)
	assert.NilError(t, err)

	targetSubDocument, err = tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)

	expectedDoc := common.NewSubDocument(srcBytes, &srcVersion2, &srcState, &srcUpdatedTime, nil, nil)
	err = targetSubDocument.Equals(expectedDoc)
	assert.NilError(t, err)

	// ==== delete a doc ====
	err = tdbclient.DeleteSubDocument(cpeMac, groupId)
	assert.NilError(t, err)

	_, err = tdbclient.GetSubDocument(cpeMac, groupId)
	assert.Assert(t, tdbclient.IsDbNotFound(err))
}

func TestDbReadDocument(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// ==== verify starting empty ====
	fields := log.Fields{}
	_, err := tdbclient.GetDocument(cpeMac)
	assert.Assert(t, tdbclient.IsDbNotFound(err))

	// ==== insert 2 docs ====
	// doc 1
	pgroupId := "privatessid"
	psrcBytes := []byte("hello world")
	psrcVersion := util.GetMurmur3Hash(psrcBytes)
	psrcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	psrcState := common.PendingDownload
	pdoc := common.NewSubDocument(psrcBytes, &psrcVersion, &psrcState, &psrcUpdatedTime, nil, nil)
	err = tdbclient.SetSubDocument(cpeMac, pgroupId, pdoc, fields)
	assert.NilError(t, err)

	// doc 2
	hgroupId := "homessid"
	hsrcBytes := []byte("red white blue")
	hsrcVersion := util.GetMurmur3Hash(hsrcBytes)
	hsrcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	hsrcState := common.PendingDownload
	hdoc := common.NewSubDocument(hsrcBytes, &hsrcVersion, &hsrcState, &hsrcUpdatedTime, nil, nil)
	err = tdbclient.SetSubDocument(cpeMac, hgroupId, hdoc, fields)
	assert.NilError(t, err)

	// ==== call ReadDocument() and verify docs from the Document are the same as the sources ====
	Document, err := tdbclient.GetDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, Document.Length(), 2)

	err = pdoc.Equals(Document.SubDocument("privatessid"))
	assert.NilError(t, err)
	err = hdoc.Equals(Document.SubDocument("homessid"))
	assert.NilError(t, err)

	// ==== delete all SubDocuments ====
	err = tdbclient.DeleteDocument(cpeMac)
	assert.NilError(t, err)

	// verify empty
	_, err = tdbclient.GetDocument(cpeMac)
	assert.Assert(t, tdbclient.IsDbNotFound(err))
}
