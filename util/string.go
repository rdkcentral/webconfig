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
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

func init() {
	rand.Seed(time.Now().Unix())
}

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
