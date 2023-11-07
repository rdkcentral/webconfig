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

	tmur "github.com/twmb/murmur3"
	"gotest.tools/assert"
)

func TestMurmur3(t *testing.T) {
	bbytes := []byte(`{"beacon_detection":true,"group_updated_time":"Wed Nov 20 22:25:26 2019"}`)
	expected := "207215293136367767981683347176289025858"

	s := GetMurmur3HashByTwmb(bbytes)
	assert.Equal(t, s, expected)
}

func TestMurmur3Int32(t *testing.T) {
	bbytes := []byte(`{"beacon_detection":true,"group_updated_time":"Wed Nov 20 22:25:26 2019"}`)
	// expected := "207215293136367767981683347176289025858"

	h32 := tmur.New32()
	h32.Write(bbytes)
	v1 := h32.Sum32()
	_ = v1
}

func TestSomeMurmur3(t *testing.T) {
	s1 := GetMurmur3Hash(nil)
	assert.Equal(t, s1, "0")

	bbytes := RandomBytes(10, 20)
	s2 := GetMurmur3Hash(bbytes)
	assert.Assert(t, len(s2) > 0)
}
