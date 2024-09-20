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
	"mime/multipart"
	"net/textproto"
)

const (
	Boundary = "2xKIxjfJuErFW+hmNCwEoMoY8I+ECM9efrV6EI4efSSW9QjI"
)

var (
	MultipartContentType = fmt.Sprintf("multipart/mixed; boundary=%s", Boundary)
)

func WriteMultipartBytes(mparts []Multipart) ([]byte, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	writer.SetBoundary(Boundary)
	for _, m := range mparts {
		header := textproto.MIMEHeader{
			HeaderContentType: {HeaderApplicationMsgpack},
			"Namespace":       {m.Name},
			"Etag":            {m.Version},
		}
		p, err := writer.CreatePart(header)
		if err != nil {
			return nil, NewError(err)
		}
		p.Write(m.Bytes)
	}
	if err := writer.Close(); err != nil {
		return nil, NewError(err)
	}

	bbytes := buffer.Bytes()
	return bbytes, nil
}
