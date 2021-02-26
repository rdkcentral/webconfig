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
	"math/big"

	tmur "github.com/twmb/murmur3"
)

// my benchmark test showed this is better by a small margin
func GetMurmur3HashByTwmb(payload []byte) string {
	// New128() is the same as New128WithSeed(0)
	h128 := tmur.New128()
	h128.Write(payload)
	v1, v2 := h128.Sum128()

	bv2 := new(big.Int)
	bv2 = bv2.SetUint64(v2)
	bv2 = bv2.Lsh(bv2, 64)

	bv1 := new(big.Int)
	bv1 = bv1.SetUint64(v1)
	bv2 = bv2.Add(bv2, bv1)
	return fmt.Sprintf("%v", bv2)
}

func GetMurmur3Hash(payload []byte) string {
	h32 := tmur.New32()
	h32.Write(payload)
	v := h32.Sum32()
	return fmt.Sprintf("%v", v)
}
