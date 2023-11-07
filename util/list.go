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
	"strings"
)

func Contains(collection interface{}, element interface{}) bool {
	switch ty := element.(type) {
	case string:
		if elements, ok := collection.([]string); ok {
			for _, e := range elements {
				if e == ty {
					return true
				}
			}
		}
	case int:
		if elements, ok := collection.([]int); ok {
			for _, e := range elements {
				if e == ty {
					return true
				}
			}
		}
	case float64:
		if elements, ok := collection.([]float64); ok {
			for _, e := range elements {
				if e == ty {
					return true
				}
			}
		}
	}
	return false
}

// TODO keep it for backward compatibility in "webconfig" for now
//      plan to remove it later
func ContainsInt(data []int, x int) bool {
	for _, d := range data {
		if d == x {
			return true
		}
	}
	return false
}

func CaseInsensitiveContains(data []string, x string) bool {
	for _, d := range data {
		if strings.EqualFold(x, d) {
			return true
		}
	}
	return false
}
