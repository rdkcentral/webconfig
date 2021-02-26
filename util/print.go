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
)

func PrettyJson(input interface{}) string {
	var x interface{}
	var pretty string

	switch ty := input.(type) {
	case string:
		if err := json.Unmarshal([]byte(ty), &x); err == nil {
			if bbytes, err := json.MarshalIndent(x, "", "    "); err == nil {
				pretty = string(bbytes)
			}
		}
	case Dict, []Dict, map[interface{}]Dict, map[string]string:
		if bbytes, err := json.MarshalIndent(input, "", "    "); err == nil {
			pretty = string(bbytes)
		}
	}

	return pretty
}
