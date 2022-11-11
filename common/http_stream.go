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
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
)

var (
	headerContentLength = []byte("Content-Length: ")
	headerDate          = []byte("Date: ")
)

func writeStatusLine(buffer *bytes.Buffer, code int, scratch []byte) {
	// hard coded http 1.1
	buffer.WriteString("HTTP/1.1 ")

	if text := http.StatusText(code); text != "" {
		buffer.Write(strconv.AppendInt(scratch[:0], int64(code), 10))
		buffer.WriteByte(' ')
		buffer.WriteString(text)
		buffer.Write(CRLF)
	} else {
		// don't worry about performance
		fmt.Fprintf(buffer, "%03d status code %d\r\n", code, code)
	}
}

func BuildPayloadAsHttp(status int, header http.Header, rbytes []byte) []byte {
	buffer := new(bytes.Buffer)

	var clenBuf [10]byte
	var statusBuf [3]byte

	writeStatusLine(buffer, status, statusBuf[:])

	header.Write(buffer)

	headerDateBytes := []byte(time.Now().Format(TimeFormat))
	buffer.Write(headerDate)
	buffer.Write(headerDateBytes)
	buffer.Write(CRLF)

	headerContentLengthBytes := strconv.AppendInt(clenBuf[:0], int64(len(rbytes)), 10)
	buffer.Write(headerContentLength)
	buffer.Write(headerContentLengthBytes)
	buffer.Write(CRLFCRLF)
	buffer.Write(rbytes)
	return buffer.Bytes()
}
