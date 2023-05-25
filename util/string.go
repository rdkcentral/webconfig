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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rdkcentral/webconfig/common"
)

var (
	telemetryFields = [][]string{
		{"version", common.HeaderProfileVersion},
		{"model", common.HeaderModelName},
		{"accountId", common.HeaderAccountID},
		{"firmwareVersion", common.HeaderFirmwareVersion},
	}
)

func ToColonMac(d string) string {
	return fmt.Sprintf("%v:%v:%v:%v:%v:%v", d[:2], d[2:4], d[4:6], d[6:8], d[8:10], d[10:12])
}

func GetAuditId() string {
	u := uuid.New()
	ustr := u.String()
	uustr := strings.ReplaceAll(ustr, "-", "")
	return uustr
}

func GenerateRandomCpeMac() string {
	u := uuid.New().String()
	return strings.ToUpper(u[len(u)-12:])
}

func ValidateMac(mac string) bool {
	if len(mac) != 12 {
		return false
	}
	for _, r := range mac {
		if r < 48 || r > 70 || (r > 57 && r < 65) {
			return false
		}
	}
	return true
}

func GetTelemetryQueryString(header http.Header, mac, queryParams, partnerId string) string {
	// build the query parameters in a fixed order
	params := []string{}

	firmwareVersion := header.Get(common.HeaderFirmwareVersion)
	if strings.Contains(firmwareVersion, "PROD") {
		params = append(params, "env=PROD")
	} else if strings.Contains(firmwareVersion, "DEV") {
		params = append(params, "env=DEV")
	}

	// special handling for partner
	if len(partnerId) > 0 {
		params = append(params, fmt.Sprintf("%v=%v", "partnerId", partnerId))
	}

	for _, pairs := range telemetryFields {
		params = append(params, fmt.Sprintf("%v=%v", pairs[0], header.Get(pairs[1])))
	}

	// eval if this broadband device
	headerWanMac := header.Get(common.HeaderWanMac)
	if len(headerWanMac) > 0 {
		headerWanMac = strings.ToUpper(headerWanMac)
		params = append(params, fmt.Sprintf("estbMacAddress=%v", headerWanMac))
		if GetMacDiff(headerWanMac, mac) == 2 {
			params = append(params, fmt.Sprintf("ecmMacAddress=%v", mac))
		}
	} else {
		estbMacAddress := GetEstbMacAddress(mac)
		params = append(params, fmt.Sprintf("estbMacAddress=%v", estbMacAddress))
		params = append(params, fmt.Sprintf("ecmMacAddress=%v", mac))
	}

	ret := strings.Join(params, "&")
	if len(queryParams) > 0 && len(ret) > 0 {
		ret += "&" + queryParams
	}
	return ret
}

func GetMacDiff(wanMac, mac string) int {
	var wanMacVal, macVal int
	if x, err := strconv.ParseInt(wanMac, 16, 64); err == nil {
		wanMacVal = int(x)
	}
	if x, err := strconv.ParseInt(mac, 16, 64); err == nil {
		macVal = int(x)
	}
	return wanMacVal - macVal
}

func ValidatePokeQuery(values url.Values) (string, error) {
	// handle ?doc=xxx
	if docQueryParamStrs, ok := values["doc"]; ok {
		if len(docQueryParamStrs) > 1 {
			err := fmt.Errorf("multiple doc parameter is not allowed")
			return "", common.NewError(err)
		}

		qparams := strings.Split(docQueryParamStrs[0], ",")
		if len(qparams) > 1 {
			err := fmt.Errorf("multiple doc parameter is not allowed")
			return "", common.NewError(err)
		}

		queryStr := qparams[0]
		if !Contains(common.SupportedPokeDocs, queryStr) {
			err := fmt.Errorf("invalid query parameter: %v", queryStr)
			return "", common.NewError(err)

		}
		return queryStr, nil
	}

	// handle ?route=xxx
	if qparams, ok := values["route"]; ok {
		if len(qparams) > 1 {
			err := fmt.Errorf("multiple route parameter is not allowed")
			return "", common.NewError(err)
		}

		qparams := strings.Split(qparams[0], ",")
		if len(qparams) > 1 {
			err := fmt.Errorf("multiple route parameter is not allowed")
			return "", common.NewError(err)
		}

		queryStr := qparams[0]
		if !Contains(common.SupportedPokeRoutes, queryStr) {
			err := fmt.Errorf("invalid query parameter: %v", queryStr)
			return "", common.NewError(err)

		}
		return queryStr, nil
	}

	// return default
	return "primary", nil
}

func GetEstbMacAddress(mac string) string {
	// if the mac cannot be parsed, then return back the input
	i, err := strconv.ParseInt(mac, 16, 64)
	if err != nil {
		return mac
	}
	return fmt.Sprintf("%012X", i+2)
}

func IsValidUTF8(bbytes []byte) bool {
	str1 := string(bbytes)
	str2 := strings.ToValidUTF8(str1, "#")
	return str1 == str2
}
