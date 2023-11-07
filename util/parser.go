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
	"net/http"
	"strings"

	"github.com/rdkcentral/webconfig/common"
)

func ParseHttp(bbytes []byte) (http.Header, []byte) {
	streams := bytes.Split(bbytes, common.CRLFCRLF)
	header := make(http.Header)
	hstreams := bytes.Split(streams[0], common.CRLF)
	for _, st := range hstreams {
		line := string(st)
		index := strings.Index(line, ":")
		if index >= 0 {
			prefix := line[:index]
			suffix := line[index+1:]
			header.Add(prefix, strings.TrimSpace(suffix))
		}
	}

	if len(streams) == 1 {
		return header, nil
	}

	return header, streams[1]
}

func BuildHttp(header http.Header, bbytes []byte) []byte {
	buffer := bytes.NewBuffer(nil)
	_ = header.Write(buffer)
	streams := make([][]byte, 2)
	streams[0] = buffer.Bytes()
	streams[1] = bbytes
	return bytes.Join(streams, common.CRLF)
}
