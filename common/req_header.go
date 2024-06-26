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
	"fmt"
	"net/http"
)

type ReqHeader struct {
	http.Header
}

func NewReqHeader(header http.Header) *ReqHeader {
	return &ReqHeader{
		Header: header,
	}
}

func (h *ReqHeader) Get(k string) (string, error) {
	v := h.Header.Get(k)
	if !IsPrintable([]byte(v)) {
		return "", fmt.Errorf("header %v invalid value %v discarded", k, v)
	}
	return v, nil
}

func IsPrintable(bbytes []byte) bool {
	for _, char := range bbytes {
		// Check if the rune is outside the printable ASCII character range.
		if char < 32 || char > 126 {
			return false
		}
	}
	return true
}
