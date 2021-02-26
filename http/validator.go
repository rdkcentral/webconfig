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

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v4"
)

func Validate(w http.ResponseWriter, r *http.Request, validateBody bool) (string, string, []byte, log.Fields, error) {
	var mac, groupId string
	var bbytes []byte
	var fields log.Fields

	// check mac
	params := mux.Vars(r)
	mac = params["mac"]
	mac = strings.ToUpper(mac)
	if !util.ValidateMac(mac) {
		err := common.Http404Error{"invalid mac"}
		return mac, groupId, bbytes, fields, common.NewError(err)
	}

	// for now, only support 1 groupId
	urlGroupIds, ok := r.URL.Query()["group_id"]
	if !ok {
		err := common.Http404Error{"no group_id"}
		return mac, groupId, bbytes, fields, common.NewError(err)
	}

	if len(urlGroupIds) > 1 {
		err := common.Http404Error{"more than 1 group_id"}
		return mac, groupId, bbytes, fields, common.NewError(err)
	}

	groupId = urlGroupIds[0]
	if len(groupId) == 0 {
		err := common.Http404Error{"more than 1 group_id"}
		return mac, groupId, bbytes, fields, common.NewError(err)
	}

	// validate body
	if validateBody {
		// check content-type
		contentType := r.Header.Get("Content-type")
		if contentType != "application/msgpack" {
			err := common.Http400Error{"content-type not msgpack"}
			return mac, groupId, bbytes, fields, common.NewError(err)
		}

		xw, ok := w.(*XpcResponseWriter)
		if !ok {
			err := common.Http500Error{"responsewriter cast error"}
			return mac, groupId, bbytes, fields, common.NewError(err)
		}
		body := xw.Body()
		if len(body) == 0 {
			err := common.Http400Error{"empty body"}
			return mac, groupId, bbytes, fields, common.NewError(err)
		}
		bbytes = []byte(body)

		// validate if input is valid tr181 msgpack
		var response common.TR181Output
		if err := msgpack.Unmarshal(bbytes, &response); err != nil {
			err := common.Http400Error{"invalid msgpack data"}
			return mac, groupId, bbytes, fields, common.NewError(err)
		}

		fields = xw.Audit()
	} else {
		xw, ok := w.(*XpcResponseWriter)
		if !ok {
			err := common.Http500Error{"responsewriter cast error"}
			return mac, groupId, bbytes, fields, common.NewError(err)
		}
		fields = xw.Audit()
	}

	return mac, groupId, bbytes, fields, nil
}
