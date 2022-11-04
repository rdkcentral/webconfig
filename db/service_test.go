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

	"github.com/rdkcentral/webconfig/common"
	"gotest.tools/assert"
)

func TestHashRootVersion(t *testing.T) {
	doc := common.NewDocument(nil)
	tt := int(123)

	// if all documents have no payload/version, calculated root should be "0"
	bbytes := []byte{}
	subdoc := common.NewSubDocument(bbytes, nil, nil, &tt, nil, nil)
	doc.SetSubDocument("advsecurity", subdoc)

	bbytes = []byte{}
	subdoc = common.NewSubDocument(bbytes, nil, nil, &tt, nil, nil)
	doc.SetSubDocument("mesh", subdoc)
	root := HashRootVersion(doc.VersionMap())
	assert.Equal(t, root, "0")

	// if some documents have payload/version, calculated root becomes non "0"
	bbytes = []byte("hello world")
	version := "12345"
	subdoc = common.NewSubDocument(bbytes, &version, nil, &tt, nil, nil)
	doc.SetSubDocument("privatessid", subdoc)
	root = HashRootVersion(doc.VersionMap())
	assert.Assert(t, root != "0")
}
