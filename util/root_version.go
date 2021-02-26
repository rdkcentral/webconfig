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
	"bytes"

	"github.com/rdkcentral/webconfig/common"
)

func RootVersion(itf interface{}) string {
	var versionMap map[string]string
	switch ty := itf.(type) {
	case []common.Multipart:
		versionMap = make(map[string]string)
		for _, mpart := range ty {
			versionMap[mpart.Name] = mpart.Version
		}
	case map[string]string:
		versionMap = ty
	}

	// if len(mparts) == 0, then the murmur hash value is 0
	buffer := bytes.NewBufferString("")
	for _, v := range versionMap {
		buffer.WriteString(v)
	}
	return GetMurmur3Hash(buffer.Bytes())
}
