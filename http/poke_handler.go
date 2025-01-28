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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
)

func (s *WebconfigServer) PokeHandler(w http.ResponseWriter, r *http.Request) {
	// handler
	params := mux.Vars(r)
	mac := params["mac"]
	mac = strings.ToUpper(mac)
	if !util.ValidateMac(mac) {
		err := common.Http400Error{
			Message: "invalid mac",
		}
		Error(w, http.StatusBadRequest, err)
		return
	}

	queryParams := r.URL.Query()

	// parse and validate query param "doc"
	// /poke?doc=telemetry
	// /poke?cpe_action=true
	pokeStr, err := util.ValidatePokeQuery(queryParams)
	if err != nil {
		Error(w, http.StatusBadRequest, err)
		return
	}

	xw, ok := w.(*XResponseWriter)
	if !ok {
		err := fmt.Errorf("PokeHandler() responsewriter cast error")
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}

	fields := xw.Audit()

	// extract "metrics_agent"
	metricsAgent := "default"
	if itf, ok := fields["metrics_agent"]; ok {
		metricsAgent = itf.(string)
	}

	deviceIds := []string{
		mac,
	}
	if x, ok := r.URL.Query()["device_id"]; ok {
		elements := strings.Split(x[0], ",")
		if len(elements) > 0 {
			deviceIds = append(deviceIds, elements...)
		}
	}

	if pokeStr == "mqtt" {
		for _, deviceId := range deviceIds {
			document, err := db.BuildMqttSendDocument(s.DatabaseClient, deviceId, fields)
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

			mbytes, err := document.HttpBytes(fields)
			if err != nil {
				Error(w, http.StatusInternalServerError, common.NewError(err))
				return
			}

			_, err = s.PostMqtt(deviceId, mbytes, fields)
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

			err = db.UpdateStatesInBatch(s.DatabaseClient, deviceId, metricsAgent, fields, document.StateMap())
			if err != nil {
				Error(w, http.StatusInternalServerError, common.NewError(err))
				return
			}
		}
		WriteAcceptedResponse(w)
		return
	}

	// pokes through cpe_action API can bypass this "smart" poke
	if len(queryParams) == 0 {
		document, err := db.BuildMqttSendDocument(s.DatabaseClient, mac, fields)
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
		pendingSubdocs := []string{}
		for subdocId := range document.StateMap() {
			pendingSubdocs = append(pendingSubdocs, subdocId)
		}
		sort.Strings(pendingSubdocs)
		fields["pending_subdocs"] = strings.Join(pendingSubdocs, ".")
	}

	// handle tokens
	token := xw.Token()
	if len(token) == 0 {
		Error(w, http.StatusForbidden, nil)
		return
	}

	transactionId, err := s.Poke(r.Header, mac, token, pokeStr, fields)

	if err != nil {
		var rherr common.RemoteHttpError
		if errors.As(err, &rherr) {
			// webpa error handling
			status := rherr.StatusCode
			if rherr.StatusCode == http.StatusNotFound {
				status = 521
			}

			// parse the core message
			var tr181Res common.TR181Response
			var tr181Message string
			if err := json.Unmarshal([]byte(rherr.Message), &tr181Res); err == nil {
				if len(tr181Res.Parameters) > 0 {
					tr181Message = tr181Res.Parameters[0].Message
				}
			}
			if len(tr181Message) > 0 {
				resp := common.HttpErrorResponse{
					Status: rherr.StatusCode,
					Errors: tr181Message,
				}
				SetAuditValue(w, "response", resp)
				WriteByMarshal(w, rherr.StatusCode, resp)
			} else {
				Error(w, status, rherr)
			}
			return
		}
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}
	data := map[string]interface{}{
		"transaction_id": transactionId,
	}
	WriteOkResponse(w, data)
}
