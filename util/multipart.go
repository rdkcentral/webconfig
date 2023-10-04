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
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/vmihailenco/msgpack"
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
	mparts := []common.Multipart{}

	if len(bbytes) == 0 {
		return mparts, nil
	}

	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return nil, common.NewError(err)
	}

	breader := bytes.NewReader(bbytes)

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
			bbytes, err := io.ReadAll(p)
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

func TelemetryBytesToMultipart(telemetryBytes []byte) (common.Multipart, error) {
	mp := common.Multipart{}

	// step 1: prepare the payload
	var itf interface{}
	err := json.Unmarshal(telemetryBytes, &itf)
	if err != nil {
		return mp, err
	}

	rbytes, err := msgpack.Marshal(&itf)
	if err != nil {
		return mp, err
	}

	parameters := []common.TR181Entry{}
	entry := common.TR181Entry{
		Name:     common.TR181NameTelemetry,
		DataType: common.TR181Blob,
		Value:    string(rbytes),
	}
	parameters = append(parameters, entry)

	// step 2 convert the "parameters" parts to json and use it to calculate the "version" hash
	var version string
	if bbytes, err := json.Marshal(parameters); err != nil {
		return mp, err
	} else {
		version = GetMurmur3Hash(bbytes)
	}

	// step 3 prepare the output object
	output := common.TR181Output{
		Parameters: parameters,
	}

	// step 4: check Accept header
	obytes, err := msgpack.Marshal(output)
	if err != nil {
		return mp, err
	}

	mp = common.Multipart{
		Bytes:   obytes,
		Version: version,
		Name:    "telemetry",
	}
	return mp, nil
}
