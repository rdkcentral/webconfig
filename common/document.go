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
	"strconv"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// deviceRootDocument is use to stored the data from device headers in the GET call
// We stored it to build "new" headers during the upstream call
// *RootDocument is used to stored data read from db and used to build the "old" headers
type Document struct {
	docmap       map[string]SubDocument
	rootDocument *RootDocument
}

// TODO add support to support NewDocument([]common.Multipart)
func NewDocument(rootDocument *RootDocument) *Document {
	docmap := map[string]SubDocument{}
	return &Document{
		rootDocument: rootDocument,
		docmap:       docmap,
	}
}

func (d *Document) SetSubDocument(groupId string, subdoc *SubDocument) {
	d.docmap[groupId] = *subdoc
}

func (d *Document) SetSubDocuments(mparts []Multipart) {
	for _, mpart := range mparts {
		version := mpart.Version
		subdoc := NewSubDocument(mpart.Bytes, &version, nil, nil, nil, nil)
		d.SetSubDocument(mpart.Name, subdoc)
	}
}

func (d *Document) SubDocument(subdocId string) *SubDocument {
	subdoc, ok := d.docmap[subdocId]
	if !ok {
		return nil
	}
	return &subdoc
}

func (d *Document) DeleteSubDocument(groupId string) {
	delete(d.docmap, groupId)
}

func (d *Document) VersionMap() map[string]string {
	versionMap := map[string]string{}
	for k, doc := range d.docmap {
		if doc.Version() != nil {
			versionMap[k] = *doc.Version()
		}
	}
	return versionMap
}

func (d *Document) StateMap() map[string]int {
	stateMap := map[string]int{}
	for k, doc := range d.docmap {
		if doc.State() != nil {
			stateMap[k] = *doc.State()
		}
	}
	return stateMap
}

func (d *Document) Length() int {
	return len(d.docmap)
}

func (d *Document) Items() map[string]SubDocument {
	return d.docmap
}

func (d *Document) SetRootDocument(rootDocument *RootDocument) {
	d.rootDocument = rootDocument
}

func (d *Document) GetRootDocument() *RootDocument {
	return d.rootDocument
}

func (d *Document) RootVersion() string {
	if d.rootDocument == nil {
		return ""
	}
	return d.rootDocument.Version
}

// TODO
// (1) for now we only filter by state
// (2) expiry check can be included to support blaster/command subdocs
// (3) we can implement blockedSubdocIds if we want
func (d *Document) FilterForMqttSend() *Document {
	newdoc := NewDocument(d.GetRootDocument())
	for subdocId, subDocument := range d.docmap {
		if subDocument.State() != nil {
			state := *subDocument.State()
			if state > Deployed {
				newdoc.SetSubDocument(subdocId, &subDocument)
			}
		}
	}
	return newdoc
}

func (d *Document) FilterForGet(versionMap map[string]string) *Document {
	newdoc := NewDocument(d.GetRootDocument())

	deviceRootVersion := versionMap["root"]
	if len(deviceRootVersion) > 0 {
		if deviceRootVersion == d.RootVersion() {
			return newdoc
		}
	}

	for subdocId, subDocument := range d.docmap {
		if subDocument.Version() != nil {
			deviceSubdocVersion := versionMap[subdocId]
			version := *subDocument.Version()
			if version != deviceSubdocVersion {
				newdoc.SetSubDocument(subdocId, &subDocument)
			}
		}
	}
	return newdoc
}

func (d *Document) Bytes() ([]byte, error) {
	if len(d.docmap) == 0 {
		return nil, nil
	}

	// build the http stream
	mparts := []Multipart{}
	for subdocId, subdoc := range d.docmap {
		mpart := Multipart{
			Bytes:   subdoc.Payload(),
			Version: *subdoc.Version(),
			Name:    subdocId,
		}
		mparts = append(mparts, mpart)
	}

	bbytes, err := WriteMultipartBytes(mparts)
	if err != nil {
		return nil, NewError(err)
	}

	return bbytes, nil
}

func (d *Document) HttpBytes(fields log.Fields) ([]byte, error) {
	// build the http stream
	mparts := []Multipart{}
	for subdocId, subdoc := range d.docmap {
		mpart := Multipart{
			Bytes:   subdoc.Payload(),
			Version: *subdoc.Version(),
			Name:    subdocId,
		}
		mparts = append(mparts, mpart)
	}

	var rootVersion string
	if d.GetRootDocument() != nil {
		rootVersion = d.RootVersion()
	} else {
		rootVersion = strconv.Itoa(int(time.Now().Unix()))
	}

	header := make(http.Header)
	header.Set(HeaderContentType, MultipartContentType)
	header.Set("Etag", rootVersion)

	var traceId string
	if itf, ok := fields["trace_id"]; ok {
		traceId = itf.(string)
	}
	if len(traceId) == 0 {
		traceId = uuid.New().String()
	}
	t := time.Now().UnixNano() / 1000
	transactionId := fmt.Sprintf("%s_____%015x", traceId, t)
	appName := fields["app_name"]
	xmoney := fmt.Sprintf("trace-id=%s;parent-id=0;span-id=0;span-name=%s", traceId, appName)
	header.Set("Transaction-Id", transactionId)
	header.Set("X-Webpa-Transaction-Id", transactionId)
	header.Set("X-Moneytrace", xmoney)

	bbytes, err := WriteMultipartBytes(mparts)
	if err != nil {
		return nil, NewError(err)
	}

	return BuildPayloadAsHttp(http.StatusOK, header, bbytes), nil
}
