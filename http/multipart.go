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
package http

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	Boundary           = "2xKIxjfJuErFW+hmNCwEoMoY8I+ECM9efrV6EI4efSSW9QjI"
	Linebreak          = '\n'
	MsgpackContentType = "Content-type: application/msgpack\r\n"
)

var (
	FirstLineBoundary    = fmt.Sprintf("--%s\r\n", Boundary)
	LineBoundary         = fmt.Sprintf("\r\n--%s\r\n", Boundary)
	LastLineBoundary     = fmt.Sprintf("\r\n--%s--\r\n", Boundary)
	MultipartContentType = fmt.Sprintf("multipart/mixed; boundary=%s", Boundary)
)

type MultipartOutput struct {
	mparts      []common.Multipart
	rootVersion string
}

func NewMultipartOutput(mparts []common.Multipart, rootVersion string) *MultipartOutput {
	return &MultipartOutput{
		mparts:      mparts,
		rootVersion: rootVersion,
	}
}

func (o *MultipartOutput) Mparts() []common.Multipart {
	return o.mparts
}

func (o *MultipartOutput) RootVersion() string {
	return o.rootVersion
}

func WriteMultipartResponse(w http.ResponseWriter, r *http.Request, o *MultipartOutput) {
	w.Header().Set("Content-type", MultipartContentType)
	w.Header().Set("Etag", o.rootVersion)
	w.WriteHeader(http.StatusOK)

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	writer.SetBoundary(Boundary)
	for _, m := range o.mparts {
		header := textproto.MIMEHeader{
			"Content-type": {"application/msgpack"},
			"Namespace":    {m.Name},
			"Etag":         {m.Version},
		}
		p, err := writer.CreatePart(header)
		if err != nil {
			panic(err)
		}
		p.Write(m.Bytes)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}

	bbytes := buffer.Bytes()
	w.Write(bbytes)
}

func (s *WebconfigServer) MultipartConfigHandler(w http.ResponseWriter, r *http.Request) {
	c := s.DatabaseClient

	// ==== data integrity check ====
	params := mux.Vars(r)
	mac, ok := params["mac"]
	if !ok || len(mac) != 12 {
		Error(w, r, http.StatusNotFound, nil)
		return
	}

	// ==== processing ====
	var fields log.Fields
	if xw, ok := w.(*XpcResponseWriter); ok {
		fields = xw.Audit()
	} else {
		err := fmt.Errorf("MultipartConfigHandler() responsewriter cast error")
		Error(w, r, http.StatusInternalServerError, err)
		return
	}

	ifNoneMatch := r.Header.Get(common.HeaderIfNoneMatch)
	supportedDocs := r.Header.Get(common.HeaderSupportedDocs)

	// NOTE that it is ok to have no "group_id". It would be the case of factory reset
	clientVersionMap := make(map[string]string)

	if qGroupIds, ok := r.URL.Query()["group_id"]; ok {
		queryStr := qGroupIds[0]
		subdocIds := strings.Split(queryStr, ",")

		versions := strings.Split(ifNoneMatch, ",")

		if len(subdocIds) != len(versions) {
			Error(w, r, http.StatusBadRequest, fmt.Errorf("group_id=%v  IF-NONE-MATCH=%v", queryStr, ifNoneMatch))
			return
		}

		for i, subdocId := range subdocIds {
			clientVersionMap[subdocId] = versions[i]
		}
	}

	// in xpcdb, all data are stored with uppercased cpemac
	mac = strings.ToUpper(mac)

	// ==== read the root version from db ====
	var rootVersion string
	rdoc, err := c.GetRootDocument(mac)

	// it is ok if err is sql.ErrNoRows, just continue the execution
	if err != nil {
		if !s.IsDbNotFound(err) {
			Error(w, r, http.StatusInternalServerError, err)
			return
		}
	} else {
		rootVersion = rdoc.Version()
		if len(rootVersion) > 0 {
			if queryRootVersion, ok := clientVersionMap["root"]; ok {
				if queryRootVersion == rootVersion {
					Error(w, r, http.StatusNotModified, nil)
					return
				}
			}
		}
	}

	folder, err := c.GetFolder(mac, fields)
	if err != nil {
		if s.IsDbNotFound(err) {
			// in the case of 404, parse and store the bitmap
			if len(supportedDocs) > 0 {
				bitmap, err := util.GetCpeBitmap(supportedDocs)
				if err != nil {
					log.WithFields(fields).Warn(common.NewError(err))
				}

				// even in 404, the bitmap could still change
				if rdoc != nil {
					if bitmap != rdoc.Bitmap() {
						err = s.SetRootDocumentBitmap(mac, bitmap)
						if err != nil {
							log.WithFields(fields).Warn(common.NewError(err))
						}
					}
				} else {
					err = s.SetRootDocumentBitmap(mac, bitmap)
					if err != nil {
						log.WithFields(fields).Warn(common.NewError(err))
					}
				}
			}
			Error(w, r, http.StatusNotFound, nil)
			return
		} else {
			Error(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	if folder.Length() == 0 {
		Error(w, r, http.StatusNotFound, nil)
		return
	}

	// parse and store x-system-supported-docs in headers
	// if errors, log warnings but do not stop
	if len(supportedDocs) > 0 {
		bitmap, err := util.GetCpeBitmap(supportedDocs)
		if err != nil {
			log.WithFields(fields).Warn(common.NewError(err))
		}

		if bitmap != rdoc.Bitmap() {
			err = s.SetRootDocumentBitmap(mac, bitmap)

			if err != nil {
				log.WithFields(fields).Warn(common.NewError(err))
			}
		}
	}

	// factory reset handling
	if s.FactoryResetEnabled() && ifNoneMatch == "NONE" {
		err := db.FactoryReset(c, mac, fields)
		if err != nil {
			if s.IsDbNotFound(err) {
				Error(w, r, http.StatusNotFound, nil)
				return
			} else {
				Error(w, r, http.StatusInternalServerError, err)
				return
			}
		}
		WriteFactoryResetResponse(w)
		return
	}

	mparts := []common.Multipart{}
	for subdocId, doc := range folder.Items() {
		clientVersion := clientVersionMap[subdocId]
		var cloudVersion string
		if doc.Version() != nil {
			cloudVersion = *doc.Version()
		}
		if cloudVersion != "" && cloudVersion == clientVersion {
			// match do nothing
		} else if len(doc.Bytes()) == 0 {
			// empty subdoc do nothing
		} else {
			mpart := common.Multipart{
				Bytes:   doc.Bytes(),
				Version: *doc.Version(),
				Name:    subdocId,
			}
			mparts = append(mparts, mpart)
		}
	}

	if len(mparts) > 0 {
		// if get no root version from db, must update db with a proper rootVersion
		if rootVersion == "" {
			rootVersion = util.RootVersion(mparts)
			if err := c.SetRootDocumentVersion(mac, rootVersion); err != nil {
				Error(w, r, http.StatusInternalServerError, err)
				return
			}
		}

		o := MultipartOutput{
			mparts:      mparts,
			rootVersion: rootVersion,
		}

		WriteMultipartResponse(w, r, &o)
	} else {
		// corner case when there are data from dao.DocumentMap() but none of them in the common.GroupSubdocMap
		Error(w, r, http.StatusNotModified, nil)
		return
	}
}
