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
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
)

func (s *WebconfigServer) Validate(w http.ResponseWriter, r *http.Request, validateContent bool) (string, string, []byte, log.Fields, error) {
	var fields log.Fields

	// check mac
	params := mux.Vars(r)
	mac := params["mac"]
	subdocId := params["subdoc_id"]
	mac = strings.ToUpper(mac)
	if s.ValidateMacEnabled() {
		if !util.ValidateMac(mac) {
			err := *common.NewHttp400Error("invalid mac")
			return mac, subdocId, nil, nil, common.NewError(err)
		}
	}

	// check for safety, but it should not fail
	xw, ok := w.(*XResponseWriter)
	if !ok {
		err := *common.NewHttp500Error("responsewriter cast error")
		return mac, subdocId, nil, nil, common.NewError(err)
	}
	fields = xw.Audit()

	if !validateContent {
		return mac, subdocId, nil, fields, nil
	}

	// ==== validate content ====
	// check content-type
	contentType := r.Header.Get("Content-type")
	if contentType != "application/msgpack" {
		// TODO (1) if we should validate this header
		//      (2) if unexpected, return 400 or 415
		err := *common.NewHttp400Error("content-type not msgpack")
		return mac, subdocId, nil, nil, common.NewError(err)
	}

	bodyBytes := xw.BodyBytes()
	if len(bodyBytes) == 0 {
		err := *common.NewHttp400Error("empty body")
		return mac, subdocId, nil, nil, common.NewError(err)
	}
	return mac, subdocId, bodyBytes, fields, nil
}
