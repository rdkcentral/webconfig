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
	"testing"
)

func TestUtilPrettyPrint(t *testing.T) {
	line := `{"foo":"bar", "enabled": true, "age": 30}`
	t.Logf(PrettyJson(line))

	a := Dict{
		"broadcast": true,
		"rindex":    10100,
		"enabled":   true,
		"sindex":    10101,
		"name":      "hello",
		"mode":      4,
		"word":      "password1",
	}
	t.Logf(PrettyJson(a))

	b := []Dict{
		Dict{
			"broadcast": true,
			"rindex":    10000,
			"enabled":   true,
			"sindex":    10001,
			"name":      "ssid_2g",
			"mode":      4,
			"word":      "password2",
		},
		Dict{
			"broadcast": true,
			"rindex":    10100,
			"enabled":   true,
			"sindex":    10101,
			"name":      "ssid_5g",
			"mode":      4,
			"word":      "password5",
		},
	}
	t.Logf(PrettyJson(b))
}
