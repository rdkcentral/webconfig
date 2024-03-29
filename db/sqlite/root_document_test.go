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

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestRootDocumentDb(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// verify starting empty
	_, err := tdbclient.GetRootDocument(cpeMac)
	assert.Assert(t, tdbclient.IsDbNotFound(err))

	// add version1 and bitmap1
	version1 := "indigo violet"
	err = tdbclient.SetRootDocumentVersion(cpeMac, version1)
	assert.NilError(t, err)

	bitmap1 := 123
	err = tdbclient.SetRootDocumentBitmap(cpeMac, bitmap1)
	assert.NilError(t, err)

	// read from db and verify identical to the sources
	rdoc, err := tdbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, version1, rdoc.Version)
	assert.Equal(t, bitmap1, rdoc.Bitmap)

	// update version
	version2 := "red white blue"
	err = tdbclient.SetRootDocumentVersion(cpeMac, version2)
	assert.NilError(t, err)

	rdoc, err = tdbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, version2, rdoc.Version)
	assert.Equal(t, bitmap1, rdoc.Bitmap)

	// update bitmap
	bitmap2 := 456
	err = tdbclient.SetRootDocumentBitmap(cpeMac, bitmap2)
	assert.NilError(t, err)

	rdoc, err = tdbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, version2, rdoc.Version)
	assert.Equal(t, bitmap2, rdoc.Bitmap)

	// set by a RootDocument
	version4 := "indigo violet"
	bitmap4 := 67
	rdoc4 := common.NewRootDocument(bitmap4, "", "", "", "", version4, "")
	err = tdbclient.SetRootDocument(cpeMac, rdoc4)
	assert.NilError(t, err)
	fetched, err := tdbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.DeepEqual(t, rdoc4.Version, fetched.Version)
	assert.DeepEqual(t, rdoc4.Bitmap, fetched.Bitmap)

	// ==== delete the root version ====
	err = tdbclient.DeleteRootDocument(cpeMac)
	assert.NilError(t, err)

	_, err = tdbclient.GetRootDocument(cpeMac)
	assert.Assert(t, tdbclient.IsDbNotFound(err))

	// ==== test delete root document version ====
	version3 := "green yellow"
	err = tdbclient.SetRootDocumentVersion(cpeMac, version3)
	assert.NilError(t, err)

	bitmap3 := 789
	err = tdbclient.SetRootDocumentBitmap(cpeMac, bitmap3)
	assert.NilError(t, err)

	err = tdbclient.DeleteRootDocumentVersion(cpeMac)
	assert.NilError(t, err)

	rdoc, err = tdbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.Equal(t, rdoc.Version, "")
	assert.Equal(t, rdoc.Bitmap, bitmap3)
}

func TestGetRootVersion(t *testing.T) {
	doc := common.NewDocument(nil)

	t1 := 123

	// if all documents have no payload/version, calculated root should be "0"
	bbytes1 := []byte{}
	d1 := common.NewSubDocument(bbytes1, nil, nil, &t1, nil, nil)
	doc.SetSubDocument("advsecurity", d1)

	bbytes2 := []byte{}
	t2 := 456
	d2 := common.NewSubDocument(bbytes2, nil, nil, &t2, nil, nil)
	doc.SetSubDocument("mesh", d2)

	root := db.HashRootVersion(doc.VersionMap())
	assert.Equal(t, root, "0")

	// if some documents have payload/version, calculated root becomes non "0"
	bbytes3 := []byte("hello world")
	version3 := "12345"
	t3 := 789
	d3 := common.NewSubDocument(bbytes3, &version3, nil, &t3, nil, nil)
	doc.SetSubDocument("privatessid", d3)

	root = db.HashRootVersion(doc.VersionMap())
	assert.Assert(t, root != "0")
}

func TestRootDocumentUpdate(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()

	// verify starting empty
	_, err := tdbclient.GetRootDocument(cpeMac)
	assert.Assert(t, tdbclient.IsDbNotFound(err))

	// ==== step 1 set a new rootdoc ====
	bitmap1 := 123
	version1 := "foo"
	schemaVersion1 := "33554433-1.3,33554434-1.3"
	modelName1 := "TG4482"
	partnerId1 := ""
	firmwareVersion1 := "TG4482PC2_4.12p7s3_PROD_sey"
	srcRootdoc1 := common.NewRootDocument(bitmap1, firmwareVersion1, modelName1, partnerId1, schemaVersion1, version1, "")

	err = tdbclient.SetRootDocument(cpeMac, srcRootdoc1)
	assert.NilError(t, err)

	tgtRootdoc1, err := tdbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.DeepEqual(t, srcRootdoc1, tgtRootdoc1)

	// ==== step 2 set the rootdoc again ====
	bitmap2 := 123
	version2 := "bar"
	schemaVersion2 := ""
	modelName2 := "TG4482"
	partnerId2 := "cox"
	firmwareVersion2 := "TG4482PC2_4.14p7s3_PROD_sey"
	rootdoc2 := common.NewRootDocument(bitmap2, firmwareVersion2, modelName2, partnerId2, schemaVersion2, version2, "")

	err = tdbclient.SetRootDocument(cpeMac, rootdoc2)
	assert.NilError(t, err)

	// ==== step 3 get the rootdoc to verify ====
	bitmap3 := 123
	version3 := "bar"
	schemaVersion3 := "33554433-1.3,33554434-1.3"
	modelName3 := "TG4482"
	partnerId3 := "cox"
	firmwareVersion3 := "TG4482PC2_4.14p7s3_PROD_sey"
	rootdoc3 := common.NewRootDocument(bitmap3, firmwareVersion3, modelName3, partnerId3, schemaVersion3, version3, "")

	tgtRootdoc3, err := tdbclient.GetRootDocument(cpeMac)
	assert.NilError(t, err)
	assert.DeepEqual(t, tgtRootdoc3, rootdoc3)
}
