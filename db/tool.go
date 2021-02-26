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
package db

import (
	"bytes"
	"fmt"
)

func GetValuesStr(length int) string {
	buffer := bytes.NewBufferString("?")
	for i := 0; i < length-1; i++ {
		buffer.WriteString(",?")
	}
	return buffer.String()
}

func GetColumnsStr(columns []string) string {
	buffer := bytes.NewBuffer([]byte{})
	for i, v := range columns {
		if i > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(v)
	}
	return buffer.String()
}

func GetSetColumnsStr(columns []string) string {
	buffer := bytes.NewBuffer([]byte{})
	for i, c := range columns {
		if i > 0 {
			buffer.WriteString(",")
		}
		s := fmt.Sprintf("%v=?", c)
		buffer.WriteString(s)
	}
	return buffer.String()
}
