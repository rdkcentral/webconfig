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
package db

import (
	"testing"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestDocumentDb(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "privatessid"

	// verify starting empty
	fields := log.Fields{}
	_, err := dbclient.GetDocument(cpeMac, groupId, fields)
	assert.Assert(t, dbclient.IsDbNotFound(err))

	// ==== insert a doc ====
	srcBytes := []byte("hello world")
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := time.Now().UnixNano() / 1000000
	srcState := common.PendingDownload
	sourceDoc := common.NewDocument(srcBytes, nil, &srcVersion, &srcState, &srcUpdatedTime)
	err = dbclient.SetDocument(cpeMac, groupId, sourceDoc, fields)
	assert.NilError(t, err)

	// read a document from db and verify identical
	targetDocument, err := dbclient.GetDocument(cpeMac, groupId, fields)
	assert.NilError(t, err)
	err = sourceDoc.Equals(targetDocument)
	assert.NilError(t, err)

	// ==== update an existing doc with the same cpeMac and groupId ====
	srcVersion2 := "red white blue"
	sourceDoc2 := common.NewDocument(nil, nil, &srcVersion2, nil, nil)
	err = dbclient.SetDocument(cpeMac, groupId, sourceDoc2, fields)
	assert.NilError(t, err)

	targetDocument, err = dbclient.GetDocument(cpeMac, groupId, fields)
	assert.NilError(t, err)

	expectedDoc := common.NewDocument(srcBytes, nil, &srcVersion2, &srcState, &srcUpdatedTime)
	err = targetDocument.Equals(expectedDoc)
	assert.NilError(t, err)

	// ==== delete a doc ====
	err = dbclient.DeleteDocument(cpeMac, groupId, fields)
	assert.NilError(t, err)

	_, err = dbclient.GetDocument(cpeMac, groupId, fields)
	assert.Assert(t, dbclient.IsDbNotFound(err))
}

func TestDbGetFolder(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// ==== verify starting empty ====
	fields := log.Fields{}
	_, err := dbclient.GetFolder(cpeMac, fields)
	assert.Assert(t, dbclient.IsDbNotFound(err))

	// ==== insert 2 docs ====
	// doc 1
	pgroupId := "privatessid"
	psrcBytes := []byte("hello world")
	psrcVersion := util.GetMurmur3Hash(psrcBytes)
	psrcUpdatedTime := time.Now().UnixNano() / 1000000
	psrcState := common.PendingDownload
	pdoc := common.NewDocument(psrcBytes, nil, &psrcVersion, &psrcState, &psrcUpdatedTime)
	err = dbclient.SetDocument(cpeMac, pgroupId, pdoc, fields)
	assert.NilError(t, err)

	// doc 2
	hgroupId := "homessid"
	hsrcBytes := []byte("red white blue")
	hsrcVersion := util.GetMurmur3Hash(hsrcBytes)
	hsrcUpdatedTime := time.Now().UnixNano() / 1000000
	hsrcState := common.PendingDownload
	hdoc := common.NewDocument(hsrcBytes, nil, &hsrcVersion, &hsrcState, &hsrcUpdatedTime)
	err = dbclient.SetDocument(cpeMac, hgroupId, hdoc, fields)
	assert.NilError(t, err)

	// ==== call GetFolder() and verify docs from the folder are the same as the sources ====
	folder, err := dbclient.GetFolder(cpeMac, fields)
	assert.NilError(t, err)
	assert.Equal(t, folder.Length(), 2)

	err = pdoc.Equals(folder.Document("privatessid"))
	assert.NilError(t, err)
	err = hdoc.Equals(folder.Document("homessid"))
	assert.NilError(t, err)

	// ==== delete all documents ====
	err = dbclient.DeleteFullDocument(cpeMac, fields)
	assert.NilError(t, err)

	// verify empty
	_, err = dbclient.GetFolder(cpeMac, fields)
	assert.Assert(t, dbclient.IsDbNotFound(err))
}
