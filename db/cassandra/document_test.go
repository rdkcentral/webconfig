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
	"strconv"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

/*
type MockGenerator struct {
    mpdict map[string]common.Multipart
}
type Multipart struct {
    Bytes   []byte
    Version string
    Name    string
}
	mparts, err := gen.GetMockMultiparts(queryStr)
*/

func getMockMultiparts(queryStr string) []common.Multipart {
	groupIds := strings.Split(queryStr, ",")
	mparts := []common.Multipart{}
	for _, g := range groupIds {
		slen := util.RandomInt(100) + 16
		bbytes := make([]byte, slen)
		rand.Read(bbytes)
		mpart := common.Multipart{
			Bytes:   bbytes,
			Version: strconv.Itoa(util.RandomInt(100000000)),
			Name:    g,
		}
		mparts = append(mparts, mpart)
	}
	return mparts
}

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
	mparts := getMockMultiparts(queryStr)
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
		// XPC-10880 enforce "params" to be non-empty
		srcDoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, nil, nil)
		srcmap[groupId] = *srcDoc

		err := tdbclient.SetSubDocument(cpeMac, groupId, srcDoc, fields)
		assert.NilError(t, err)

		fetchedDoc, err := tdbclient.GetSubDocument(cpeMac, groupId)
		assert.NilError(t, err)

		// assert.Assert(t, srcDoc.Equals(fetchedDoc))
		err = srcDoc.Equals(fetchedDoc)
		// assert.Assert(t, okok)
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
}
