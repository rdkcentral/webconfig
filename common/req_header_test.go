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
	"net/http"
	"testing"

	"gotest.tools/assert"
)

func TestIsPrintable(t *testing.T) {
	b1 := []byte("hello world")
	assert.Assert(t, IsPrintable(b1))
	b2 := []byte{0x00, 0x00, 0x00, 0x01}
	assert.Assert(t, !IsPrintable(b2))
	b3 := append(b1, b2...)
	assert.Assert(t, !IsPrintable(b3))

	b4 := []byte("CGM4140COM_6.8p8s1_PROD_sey")
	b5 := []byte{0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8}
	b6 := append(b4, b5...)
	assert.Assert(t, !IsPrintable(b6))
}

func TestReqHeader(t *testing.T) {
	s1 := "hello world"
	s2 := string([]byte{0x00, 0x00, 0x00, 0x01})
	s3 := s1 + s2

	header := make(http.Header)
	k1 := "maroon"
	header.Set(k1, "helloworld")
	k2 := "auburn"
	header.Set(k2, s2)
	k3 := "amber"
	header.Set(k3, s3)
	reqHeader := NewReqHeader(header)

	v1, err := reqHeader.Get(k1)
	assert.NilError(t, err)
	assert.Equal(t, v1, "helloworld")

	v2, err := reqHeader.Get(k2)
	assert.Assert(t, err != nil)
	assert.Equal(t, v2, "")

	v3, err := reqHeader.Get(k3)
	assert.Assert(t, err != nil)
	assert.Equal(t, v3, "")

	v4, err := reqHeader.Get("viridian")
	assert.NilError(t, err)
	assert.Equal(t, v4, "")
}
