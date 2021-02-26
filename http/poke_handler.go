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

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"github.com/gorilla/mux"
)

func (s *WebconfigServer) PokeHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	mac, ok := params["mac"]
	if !util.ValidateMac(mac) {
		err := common.Http404Error{"invalid mac"}
		Error(w, r, http.StatusNotFound, err)
		return
	}

	xw, ok := w.(*XpcResponseWriter)
	if !ok {
		err := fmt.Errorf("PokeHandler() responsewriter cast error")
		Error(w, r, http.StatusInternalServerError, common.NewError(err))
		return
	}

	token := xw.Token()
	fields := xw.Audit()
	var err error

	if len(token) == 0 {
		token, err = s.GetToken(fields)
		if err != nil {
			Error(w, r, http.StatusInternalServerError, common.NewError(err))
			return
		}
	}

	transactionId, err := s.Poke(mac, token, fields)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.As(err, common.RemoteHttpErrorType) {
			unerr := errors.Unwrap(err)
			rherr := unerr.(common.RemoteHttpError)

			// webpa error handling
			if rherr.StatusCode == http.StatusNotFound {
				status = 520
			} else if rherr.StatusCode > http.StatusInternalServerError {
				status = rherr.StatusCode

			}
		}
		Error(w, r, status, err)
		return
	}
	data := map[string]interface{}{
		"transaction_id": transactionId,
	}
	WriteOkResponse(w, r, data)
}
