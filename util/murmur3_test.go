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
	"bytes"
	"testing"

	tmur "github.com/twmb/murmur3"
)

func TestMurmur3(t *testing.T) {
	bbytes := []byte(`{"enabled":true,"updated_time":"Fri Jan 01 00:00:00 2021"}`)
	s := GetMurmur3HashByTwmb(bbytes)
	t.Logf("s=%v", s)
}

func TestMurmur3Int32(t *testing.T) {
	bbytes := []byte(`{"enabled":true,"updated_time":"Fri Jan 01 00:00:00 2021"}`)

	h32 := tmur.New32()
	h32.Write(bbytes)
	v1 := h32.Sum32()
	t.Logf("v1=%v  type(v1)=%T\n", v1, v1)
}

func TestSomeMurmur3(t *testing.T) {
	buffer := bytes.NewBufferString("")
	bbytes := buffer.Bytes()
	t.Logf("\n|%v|\n", string(bbytes))

	x := GetMurmur3Hash(bbytes)
	t.Logf("\nx=|%v|\n", x)
}
