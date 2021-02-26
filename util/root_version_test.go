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
package util

import (
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"gotest.tools/assert"
)

func TestGetRootVersion(t *testing.T) {
	folder := common.NewFolder()

	t1 := int64(123)

	// if all documents have no payload/version, calculated root should be "0"
	bbytes1 := []byte{}
	params1 := "foo bar"
	d1 := common.NewDocument(bbytes1, &params1, nil, nil, &t1)
	folder.SetDocument("advsecurity", d1)

	bbytes2 := []byte{}
	params2 := "hello world"
	t2 := int64(456)
	d2 := common.NewDocument(bbytes2, &params2, nil, nil, &t2)
	folder.SetDocument("mesh", d2)

	root := RootVersion(folder.VersionMap())
	assert.Equal(t, root, "0")

	// if some documents have payload/version, calculated root becomes non "0"
	bbytes3 := []byte("hello world")
	params3 := "red blue white"
	version3 := "12345"
	t3 := int64(789)
	d3 := common.NewDocument(bbytes3, &params3, &version3, nil, &t3)
	folder.SetDocument("privatessid", d3)

	root = RootVersion(folder.VersionMap())
	assert.Assert(t, root != "0")
}
