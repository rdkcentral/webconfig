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
package common

import (
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestRootDocumentCompare(t *testing.T) {
	bitmap := 123
	version := "foo"
	schemaVersion := "33554433-1.3,33554434-1.3"
	modelName := "bar"
	partnerId := "cox"
	firmwareVersion := "TG4482PC2_4.12p7s3_PROD_sey"
	rootdoc1 := NewRootDocument(bitmap, firmwareVersion, modelName, partnerId, schemaVersion, version, "")
	rootdoc2 := rootdoc1.Clone()
	enum := rootdoc1.Compare(rootdoc2)
	assert.Equal(t, enum, RootDocumentEquals)

	firmwareVersion3 := "TG4482PC2_4.14p7s3_PROD_sey"
	rootdoc3 := NewRootDocument(bitmap, firmwareVersion3, modelName, partnerId, schemaVersion, version, "")
	enum = rootdoc1.Compare(rootdoc3)
	assert.Equal(t, enum, RootDocumentMetaChanged)

	version4 := "3456"
	rootdoc4 := NewRootDocument(bitmap, firmwareVersion, modelName, partnerId, schemaVersion, version4, "")
	enum = rootdoc1.Compare(rootdoc4)
	assert.Equal(t, enum, RootDocumentVersionOnlyChanged)
}

func TestRootDocumentUpdate(t *testing.T) {
	bitmap1 := 123
	version1 := "foo"
	schemaVersion1 := "33554433-1.3,33554434-1.3"
	modelName1 := "TG4482"
	partnerId1 := ""
	firmwareVersion1 := "TG4482PC2_4.12p7s3_PROD_sey"
	rootdoc1 := NewRootDocument(bitmap1, firmwareVersion1, modelName1, partnerId1, schemaVersion1, version1, "")

	bitmap2 := 123
	version2 := "bar"
	schemaVersion2 := ""
	modelName2 := "TG4482"
	partnerId2 := "cox"
	firmwareVersion2 := "TG4482PC2_4.14p7s3_PROD_sey"
	rootdoc2 := NewRootDocument(bitmap2, firmwareVersion2, modelName2, partnerId2, schemaVersion2, version2, "")

	bitmap3 := 123
	version3 := "bar"
	schemaVersion3 := "33554433-1.3,33554434-1.3"
	modelName3 := "TG4482"
	partnerId3 := "cox"
	firmwareVersion3 := "TG4482PC2_4.14p7s3_PROD_sey"
	rootdoc3 := NewRootDocument(bitmap3, firmwareVersion3, modelName3, partnerId3, schemaVersion3, version3, "")

	rootdoc1.Update(rootdoc2)
	assert.Equal(t, *rootdoc1, *rootdoc3)
	assert.DeepEqual(t, rootdoc1, rootdoc3)

	line := rootdoc1.String()
	assert.Assert(t, !strings.Contains(line, "map["))
}

func TestRootDocumentIsDifferent(t *testing.T) {
	bitmap := 123
	version := "foo"
	schemaVersion := "33554433-1.3,33554434-1.3"
	modelName := "bar"
	partnerId := "cox"
	firmwareVersion := "TG4482PC2_4.12p7s3_PROD_sey"
	rootdoc1 := NewRootDocument(bitmap, firmwareVersion, modelName, partnerId, schemaVersion, version, "")
	rootdoc2 := rootdoc1.Clone()
	isDiff := rootdoc1.IsDifferent(rootdoc2)
	assert.Assert(t, !isDiff)

	firmwareVersion3 := "TG4482PC2_4.14p7s3_PROD_sey"
	rootdoc3 := NewRootDocument(bitmap, firmwareVersion3, modelName, partnerId, schemaVersion, version, "")
	isDiff = rootdoc1.IsDifferent(rootdoc3)
	assert.Assert(t, isDiff)
}
