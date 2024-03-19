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
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestRefSubDocumentOperation(t *testing.T) {
	refId := uuid.New().String()

	// prepare the source data
	slen := util.RandomInt(100) + 16
	srcBytes := make([]byte, slen)
	rand.Read(srcBytes)
	srcVersion := util.GetMurmur3Hash(srcBytes)

	// verify empty before start
	var err error
	_, err = tdbclient.GetRefSubDocument(refId)
	assert.Assert(t, tdbclient.IsDbNotFound(err))

	// write into db
	srcRefsubdoc := common.NewRefSubDocument(srcBytes, &srcVersion)
	err = tdbclient.SetRefSubDocument(refId, srcRefsubdoc)
	assert.NilError(t, err)

	fetchedRefsubdoc, err := tdbclient.GetRefSubDocument(refId)
	assert.NilError(t, err)
	assert.Assert(t, srcRefsubdoc.Equals(fetchedRefsubdoc))

	err = tdbclient.DeleteRefSubDocument(refId)
	assert.NilError(t, err)

	// verify not found in db now
	_, err = tdbclient.GetRefSubDocument(refId)
	assert.Assert(t, tdbclient.IsDbNotFound(err))
}
