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

	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestRootDocumentDb(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// verify starting empty
	_, err := dbclient.GetRootDocument(cpeMac)
	assert.Assert(t, dbclient.IsDbNotFound(err))

	// add version1 and bitmap1
	version1 := "indigo violet"
	err = dbclient.SetRootDocumentVersion(cpeMac, version1)
	assert.NilError(t, err)

	bitmap1 := 123
	err = dbclient.SetRootDocumentBitmap(cpeMac, bitmap1)
	assert.NilError(t, err)

	// read from db and verify identical to the sources
	rdoc, err := dbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, version1, rdoc.Version())
	assert.Equal(t, bitmap1, rdoc.Bitmap())

	// update version
	version2 := "red white blue"
	err = dbclient.SetRootDocumentVersion(cpeMac, version2)
	assert.NilError(t, err)

	rdoc, err = dbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, version2, rdoc.Version())
	assert.Equal(t, bitmap1, rdoc.Bitmap())

	// update bitmap
	bitmap2 := 456
	err = dbclient.SetRootDocumentBitmap(cpeMac, bitmap2)
	assert.NilError(t, err)

	rdoc, err = dbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, version2, rdoc.Version())
	assert.Equal(t, bitmap2, rdoc.Bitmap())

	// ==== delete the root version ====
	err = dbclient.DeleteRootDocument(cpeMac)
	assert.NilError(t, err)

	_, err = dbclient.GetRootDocument(cpeMac)
	assert.Assert(t, dbclient.IsDbNotFound(err))

	// ==== test delete root document version ====
	version3 := "green yellow"
	err = dbclient.SetRootDocumentVersion(cpeMac, version3)
	assert.NilError(t, err)

	bitmap3 := 789
	err = dbclient.SetRootDocumentBitmap(cpeMac, bitmap3)
	assert.NilError(t, err)

	err = dbclient.DeleteRootDocumentVersion(cpeMac)
	assert.NilError(t, err)

	rdoc, err = dbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, rdoc.Version(), "")
	assert.Equal(t, rdoc.Bitmap(), bitmap3)
}
