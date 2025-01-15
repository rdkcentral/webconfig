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
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestValidateMac(t *testing.T) {
	mac := "001122334455"
	assert.Assert(t, ValidateMac(mac))

	mac = "4444ABCDEF01"
	assert.Assert(t, ValidateMac(mac))

	mac = "00112233445Z"
	assert.Assert(t, !ValidateMac(mac))

	mac = "001122334455Z"
	assert.Assert(t, !ValidateMac(mac))

	mac = "0H1122334455"
	assert.Assert(t, !ValidateMac(mac))

	for i := 0; i < 10; i++ {
		mac := GenerateRandomCpeMac()
		assert.Assert(t, ValidateMac(mac))
	}
}

func TestValidatePokeQuery(t *testing.T) {
	values := url.Values{}

	values["doc"] = []string{
		"hello,world",
		"primary,telemetry",
	}
	_, err := ValidatePokeQuery(values)
	assert.Assert(t, err != nil)

	values["doc"] = []string{
		"primary,hello,world",
	}
	_, err = ValidatePokeQuery(values)
	assert.Assert(t, err != nil)

	values["doc"] = []string{
		"primary,telemetry",
	}
	_, err = ValidatePokeQuery(values)
	assert.NilError(t, err)

	values["doc"] = []string{
		"primary",
	}
	s, err := ValidatePokeQuery(values)
	assert.NilError(t, err)
	assert.Equal(t, s, "primary")

	values["doc"] = []string{
		"telemetry",
	}
	s, err = ValidatePokeQuery(values)
	assert.NilError(t, err)
	assert.Equal(t, s, "telemetry")

	delete(values, "doc")
	s, err = ValidatePokeQuery(values)
	assert.NilError(t, err)
	assert.Equal(t, s, "root")

	values["doc"] = []string{
		"primary",
	}
	values["route"] = []string{
		"mqtt",
	}
	s, err = ValidatePokeQuery(values)
	assert.NilError(t, err)
	assert.Equal(t, s, "primary")

	delete(values, "doc")
	s, err = ValidatePokeQuery(values)
	assert.NilError(t, err)
	assert.Equal(t, s, "mqtt")
}

func TestValidateQueryParams(t *testing.T) {
	cpeMac := GenerateRandomCpeMac()
	validSubdocIdMap := maps.Clone(common.SubdocBitIndexMap)
	validSubdocIdMap["red"] = 1
	validSubdocIdMap["orange"] = 1

	// case 1
	deviceConfigUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	fields := make(log.Fields)
	err = ValidateQueryParams(req, validSubdocIdMap, fields)
	assert.Assert(t, errors.Is(err, common.ErrInvalidQueryParams))

	// case 2
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?foo=bar", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	err = ValidateQueryParams(req, validSubdocIdMap, fields)
	assert.Assert(t, errors.Is(err, common.ErrInvalidQueryParams))

	// case 3
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	err = ValidateQueryParams(req, validSubdocIdMap, fields)
	assert.Assert(t, errors.Is(err, common.ErrInvalidQueryParams))

	// case 4
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	err = ValidateQueryParams(req, validSubdocIdMap, fields)
	assert.Assert(t, errors.Is(err, common.ErrInvalidQueryParams))

	// case 5
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,foo", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "123,234")
	err = ValidateQueryParams(req, validSubdocIdMap, fields)
	assert.Assert(t, errors.Is(err, common.ErrInvalidQueryParams))

	// case 6
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,privatessid,foo", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "123,234,345")
	err = ValidateQueryParams(req, validSubdocIdMap, fields)
	assert.Assert(t, errors.Is(err, common.ErrInvalidQueryParams))

	// case 7
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,privatessid,homessid", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "123,234,345,456")
	err = ValidateQueryParams(req, validSubdocIdMap, fields)
	assert.Assert(t, errors.Is(err, common.ErrInvalidQueryParams))

	// case 8
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,privatessid,homessid,lan", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "123,234,345,456")
	err = ValidateQueryParams(req, validSubdocIdMap, fields)
	assert.NilError(t, err)

	// case 9
	deviceConfigUrl = fmt.Sprintf("/api/v1/device/%v/config?group_id=root,privatessid,homessid,red,orange", cpeMac)
	req, err = http.NewRequest("GET", deviceConfigUrl, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderIfNoneMatch, "123,234,345,456,678")
	err = ValidateQueryParams(req, validSubdocIdMap, fields)
	assert.NilError(t, err)
}
