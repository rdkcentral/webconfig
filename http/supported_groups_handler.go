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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"github.com/gorilla/mux"
)

// The supported doc header in GET /config is parsed and stored as a bitmap
// this API returns the bitmap in read friendly json
func (s *WebconfigServer) GetSupportedGroupsHandler(w http.ResponseWriter, r *http.Request) {
	// check mac
	params := mux.Vars(r)
	mac := params["mac"]
	mac = strings.ToUpper(mac)
	if !util.ValidateMac(mac) {
		err := common.NewError(fmt.Errorf("invalid mac"))
		Error(w, http.StatusBadRequest, err)
		return
	}

	rdoc, err := s.GetRootDocument(mac)
	if err != nil {
		if s.IsDbNotFound(err) {
			err = errors.Unwrap(err)
			Error(w, http.StatusNotFound, err)
			return
		}
		Error(w, http.StatusInternalServerError, err)
		return
	}

	outdata := common.SupportedGroupsData{
		Bitmap: rdoc.Bitmap,
		Groups: util.GetSupportedMap(rdoc.Bitmap),
	}

	WriteOkResponse(w, outdata)
}
