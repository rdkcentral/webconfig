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
	crand "crypto/rand"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestDocument(t *testing.T) {
	document := NewDocument(nil)
	assert.Equal(t, len(document.RootVersion()), 0)

	bitmap := 123
	version := "foo"
	schemaVersion := "33554433-1.3,33554434-1.3"
	modelName := "bar"
	partnerId := "cox"
	firmwareVersion := "TG4482PC2_4.12p7s3_PROD_sey"
	rootdoc := NewRootDocument(bitmap, firmwareVersion, modelName, partnerId, schemaVersion, version, "")
	document = NewDocument(rootdoc)

	subdocIds := []string{"red", "orange", "yellow", "green"}
	mparts := []Multipart{}
	versionMap := make(map[string]string)
	for _, subdocId := range subdocIds {
		blen := rand.Intn(10) + 10
		bbytes := make([]byte, blen)
		crand.Read(bbytes)
		version := strconv.Itoa(int(time.Now().Unix()))
		mpart := Multipart{
			Bytes:   bbytes,
			Version: version,
			Name:    subdocId,
			State:   Deployed,
		}
		mparts = append(mparts, mpart)
		versionMap[subdocId] = version
	}

	document.SetSubDocuments(mparts)
	assert.Equal(t, len(document.Items()), len(subdocIds))

	filteredDocument := document.FilterForGet(versionMap)
	assert.Assert(t, filteredDocument != nil)
	assert.Equal(t, len(filteredDocument.Items()), 0)

	filteredDocument = document.FilterForGet(nil)
	assert.Assert(t, filteredDocument != nil)
	assert.Equal(t, len(filteredDocument.Items()), 4)
}
