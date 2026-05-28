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

	"gotest.tools/assert"
)

func TestRefSubDocument(t *testing.T) {
	bbytes1 := []byte("hello world")
	version1 := "12345"
	refsubdoc1 := NewRefSubDocument(bbytes1, &version1)

	bbytes2 := []byte("hello world")
	version2 := "12345"
	refsubdoc2 := NewRefSubDocument(bbytes2, &version2)
	assert.Assert(t, refsubdoc1.Equals(refsubdoc2))

	bbytes3 := []byte("foo bar")
	version3 := "12345"
	refsubdoc3 := NewRefSubDocument(bbytes3, &version3)
	assert.Assert(t, !refsubdoc1.Equals(refsubdoc3))
}
