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
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/rdkcentral/webconfig/common"
)

func ParseMultipart(header http.Header, bbytes []byte) (map[string]common.Multipart, error) {
	mpartmap := make(map[string]common.Multipart)
	mparts, err := ParseMultipartAsList(header, bbytes)
	if err != nil {
		return mpartmap, common.NewError(err)
	}
	for _, mpart := range mparts {
		mpartmap[mpart.Name] = mpart
	}
	return mpartmap, nil
}

func ParseMultipartAsList(header http.Header, bbytes []byte) ([]common.Multipart, error) {
	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		log.Fatal(err)
	}

	breader := bytes.NewReader(bbytes)

	mparts := []common.Multipart{}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(breader, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return mparts, common.NewError(err)
			}
			bbytes, err := ioutil.ReadAll(p)
			if err != nil {
				return mparts, common.NewError(err)
			}

			// build the response
			subdocId := p.Header.Get("Namespace")
			if len(subdocId) == 0 {
				continue
			}

			mpart := common.Multipart{
				Bytes:   bbytes,
				Version: p.Header.Get("Etag"),
				Name:    subdocId,
			}
			mparts = append(mparts, mpart)
		}
	}
	return mparts, nil
}
