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
	"strconv"
	"strings"

	"github.com/rdkcentral/webconfig/common"
)

// bitmap is used to name int variables
// bitarray is used to name string variables

func SetBitmapByGroup(cpeBitmap *int, groupId int, groupBitmap int) error {
	tuples, ok := common.SupportedDocsBitMaskMap[groupId]
	if !ok {
		// XPC-15313 ignore unknown groupId, instead of raising an error
		return nil
	}

	for _, tuple := range tuples {
		mask := 1 << (tuple.GroupBit - 1)
		masked := groupBitmap & mask
		if masked > 0 {
			(*cpeBitmap) |= 1 << (tuple.CpeBit - 1)
		}
	}
	return nil
}

func ParseFirmwareGroupBitarray(s string) (int, int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, 0, common.NewError(err)
	}
	groupId := i >> 24
	groupBitMask := 1<<24 - 1
	groupBitmap := i & groupBitMask
	return groupId, groupBitmap, nil
}

func GetCpeBitmap(rdkSupportedDocsHeaderStr string) (int, error) {
	cpeBitmap := 0

	sids := strings.Split(rdkSupportedDocsHeaderStr, ",")
	for _, sid := range sids {
		groupId, groupBitmap, err := ParseFirmwareGroupBitarray(sid)
		if err != nil {
			return 0, common.NewError(err)
		}

		err = SetBitmapByGroup(&cpeBitmap, groupId, groupBitmap)
		if err != nil {
			return 0, common.NewError(err)
		}
	}

	return cpeBitmap, nil
}

func PrettyBitarray(i int) string {
	x := fmt.Sprintf("%032b", i)
	return fmt.Sprintf("%s %s %s %s %s %s %s %s",
		x[0:4],
		x[4:8],
		x[8:12],
		x[12:16],
		x[16:20],
		x[20:24],
		x[24:28],
		x[28:32])
}

func PrettyGroupBitarray(i int) string {
	x := fmt.Sprintf("%032b", i)
	return fmt.Sprintf("%s %s %s %s %s %s %s",
		x[0:8],
		x[8:12],
		x[12:16],
		x[16:20],
		x[20:24],
		x[24:28],
		x[28:32])
}

func IsSubdocSupported(cpeBitmap int, subdocId string) bool {
	index, ok := common.SubdocBitIndexMap[subdocId]
	if !ok {
		return false
	}

	shift := index - 1
	bitmask := 1 << shift
	masked := cpeBitmap & bitmask
	return masked != 0
}

func GetSupportedMap(cpeBitmap int) map[string]bool {
	supportedMap := map[string]bool{}

	for k, index := range common.SubdocBitIndexMap {
		shift := index - 1
		bitmask := 1 << shift
		supportedMap[k] = false
		if masked := cpeBitmap & bitmask; masked > 0 {
			supportedMap[k] = true
		}
	}
	return supportedMap
}

func BitarrayToBitmap(src string) (int, error) {
	s := strings.ReplaceAll(src, " ", "")
	v, err := strconv.ParseInt(s, 2, 64)
	if err != nil {
		return 0, common.NewError(err)
	}
	i := int(v)
	return i, nil
}

func GetBitmapFromSupportedMap(srcMap map[string]bool) int {
	var bitmap int

	for k, index := range common.SubdocBitIndexMap {
		shift := index - 1
		bitmask := 1 << shift
		if srcMap[k] {
			bitmap += bitmask
		}
	}
	return bitmap
}
