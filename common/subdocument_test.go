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
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestSubDocumentString(t *testing.T) {
	bbytes := []byte("hello world")
	version := "789345"
	state := Failure
	updatedTime := int(time.Now().UnixNano() / 1000000)
	errorCode := 103
	errorDetails := "cannot parse"

	subdoc := NewSubDocument(bbytes, &version, &state, &updatedTime, &errorCode, &errorDetails)
	assert.Assert(t, subdoc != nil)

	subdoc = &SubDocument{}
	tgtVersion := subdoc.GetVersion()
	assert.Equal(t, tgtVersion, "")
	tgtState := subdoc.GetState()
	assert.Equal(t, tgtState, 0)
}
