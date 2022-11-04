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

	"gotest.tools/assert"
)

func TestRandomInt(t *testing.T) {
	n := 10
	for i := 0; i < 30; i++ {
		x := RandomInt(n)
		assert.Assert(t, x >= 0 && x < 10)
	}
}

func TestRandomBytes(t *testing.T) {
	for i := 0; i < 100; i++ {
		data1 := RandomBytes(7, 7)
		assert.Equal(t, len(data1), 7)

		data2 := RandomBytes(10, 20)
		assert.Assert(t, len(data2) >= 10 && len(data2) < 20)
	}
}
