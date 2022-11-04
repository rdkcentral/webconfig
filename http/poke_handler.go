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

	"github.com/gorilla/mux"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
)

func (s *WebconfigServer) PokeHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	mac := params["mac"]
	if !util.ValidateMac(mac) {
		err := common.Http404Error{
			Message: "invalid mac",
		}
		Error(w, http.StatusNotFound, err)
		return
	}

	queryParams := r.URL.Query()

	// parse and validate query param "doc"
	pokeStr, err := util.ValidatePokeQuery(queryParams)
	if err != nil {
		Error(w, http.StatusBadRequest, err)
		return
	}

	xw, ok := w.(*XpcResponseWriter)
	if !ok {
		err := fmt.Errorf("PokeHandler() responsewriter cast error")
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	token := xw.Token()
	fields := xw.Audit()

	if len(token) == 0 {
		token, err = s.GetToken(fields)
		if err != nil {
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}
	}

	// extract "metrics_agent"
	metricsAgent := "default"
	if itf, ok := fields["metrics_agent"]; ok {
		metricsAgent = itf.(string)
	}

	// XPC-15999
	var document *common.Document
	if pokeStr == "mqtt" {
		document, err = db.BuildMqttSendDocument(s.DatabaseClient, mac, fields)
		if err != nil {
			if s.IsDbNotFound(err) {
				Error(w, http.StatusNotFound, nil)
				return
			}
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}
		if document.Length() == 0 {
			WriteResponseBytes(w, nil, http.StatusNoContent)
			return
		}

		// TODO, we can build/filter it again for blocked subdocs if needed

		mbytes, err := document.Bytes()
		if err != nil {
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}

		rbytes, err := s.PostMqtt(mac, mbytes, fields)
		if err != nil {
			var rherr common.RemoteHttpError
			if errors.As(err, &rherr) {
				if rherr.StatusCode == http.StatusNotFound {
					Error(w, http.StatusNotFound, nil)
					return
				}
			}
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}

		err = db.UpdateStatesInBatch(s.DatabaseClient, mac, metricsAgent, fields, document.StateMap())
		if err != nil {
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}

		WriteResponseBytes(w, rbytes, http.StatusAccepted)
		return
	}

	// pokes through cpe_action API can bypass this "smart" poke
	_, ok = queryParams["cpe_action"]
	if !ok {
		document, err = db.BuildMqttSendDocument(s.DatabaseClient, mac, fields)
		if err != nil {
			if s.IsDbNotFound(err) {
				Error(w, http.StatusNoContent, nil)
				return
			}
			Error(w, http.StatusInternalServerError, common.NewError(err))
			return
		}
		if document.Length() == 0 {
			WriteResponseBytes(w, nil, http.StatusNoContent)
			return
		}
	}

	transactionId, err := s.Poke(mac, token, pokeStr, fields)
	if err != nil {
		status := http.StatusInternalServerError
		var rherr common.RemoteHttpError
		if errors.As(err, &rherr) {
			// webpa error handling
			if rherr.StatusCode == http.StatusNotFound {
				status = 521
			} else if rherr.StatusCode > http.StatusInternalServerError {
				status = rherr.StatusCode
			}
		}
		Error(w, status, err)
		return
	}
	data := map[string]interface{}{
		"transaction_id": transactionId,
	}
	WriteOkResponse(w, data)
}
