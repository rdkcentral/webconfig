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
	"encoding/json"
	"net/http"
	"testing"

	"gotest.tools/assert"
)

func TestUtilPrettyPrint(t *testing.T) {
	line := `{"foo":"bar", "enabled": true, "age": 30}`
	t.Logf(PrettyJson(line))

	a := Dict{
		"broadcast_ssid":  true,
		"radio_index":     10100,
		"ssid_enabled":    true,
		"ssid_index":      10101,
		"ssid_name":       "hello",
		"wifi_security":   4,
		"wifi_passphrase": "password1",
	}
	t.Logf(PrettyJson(a))

	b := []Dict{
		{
			"broadcast_ssid":  true,
			"radio_index":     10000,
			"ssid_enabled":    true,
			"ssid_index":      10001,
			"ssid_name":       "ssid_2g",
			"wifi_security":   4,
			"wifi_passphrase": "password2",
		},
		{
			"broadcast_ssid":  true,
			"radio_index":     10100,
			"ssid_enabled":    true,
			"ssid_index":      10101,
			"ssid_name":       "ssid_5g",
			"wifi_security":   4,
			"wifi_passphrase": "password5",
		},
	}
	t.Logf(PrettyJson(b))

	d := Dict{}
	err := json.Unmarshal([]byte(line), &d)
	assert.NilError(t, err)
	assert.Equal(t, len(d), 3)
	assert.Equal(t, d.GetString("xxx"), "")
}

func TestDictDefaults(t *testing.T) {
	// ==== bool ====
	d1 := Dict{
		"red": true,
	}

	b1 := d1.GetBool("red")
	assert.Equal(t, b1, true)
	b1 = d1.GetBool("red", false)
	assert.Equal(t, b1, true)

	b1 = d1.GetBool("orange")
	assert.Equal(t, b1, false)
	b1 = d1.GetBool("orange", false)
	assert.Equal(t, b1, false)
	b1 = d1.GetBool("orange", true)
	assert.Equal(t, b1, true)

	// ==== int ====
	d2 := Dict{
		"red": 123,
	}

	i2 := d2.GetInt("red")
	assert.Equal(t, i2, 123)
	i2 = d2.GetInt("red", 456)
	assert.Equal(t, i2, 123)

	i2 = d2.GetInt("orange")
	assert.Equal(t, i2, 0)
	i2 = d2.GetInt("orange", 78)
	assert.Equal(t, i2, 78)

	// ==== string ====
	d3 := Dict{
		"red": "foo",
	}

	s3 := d3.GetString("red")
	assert.Equal(t, s3, "foo")
	s3 = d3.GetString("red", "bar")
	assert.Equal(t, s3, "foo")

	s3 = d3.GetString("orange")
	assert.Equal(t, s3, "")
	s3 = d3.GetString("orange", "bar")
	assert.Equal(t, s3, "bar")

	// ==== string ====
	d4 := Dict{
		"red":    "foo",
		"orange": "",
	}
	s4 := d4.GetString("orange", "orange")
	assert.Equal(t, s4, "")

	s4 = d4.GetNonEmptyString("orange", "orange")
	assert.Equal(t, s4, "orange")
}

func TestHeaderToMap(t *testing.T) {
	header := make(http.Header)
	header.Add("Red", "maroon")
	header.Add("Orange", "auburn")
	header.Add("Yellow", "amber")

	expected := map[string]string{
		"Red":    "maroon",
		"Orange": "auburn",
		"Yellow": "amber",
	}

	m := HeaderToMap(header)
	assert.DeepEqual(t, m, expected)
}
