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

func TestContains(t *testing.T) {
	days := []string{"mon", "tue", "wed", "thu"}
	c1 := Contains(days, "wed")
	assert.Assert(t, c1)
	c2 := Contains(days, "fri")
	assert.Assert(t, !c2)
}

func TestContainsInt(t *testing.T) {
	values := []int{1, 2, 3, 4}
	c1 := ContainsInt(values, 3)
	assert.Assert(t, c1)
	c2 := ContainsInt(values, 5)
	assert.Assert(t, !c2)
}

func TestCaseInsensitiveContains(t *testing.T) {
	days := []string{"lon", "tue", "Wed", "thu"}
	c1 := CaseInsensitiveContains(days, "weD")
	assert.Assert(t, c1)
	c2 := CaseInsensitiveContains(days, "fri")
	assert.Assert(t, !c2)
}
