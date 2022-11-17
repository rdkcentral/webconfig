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
package security

import "github.com/go-akka/configuration"

const (
	Plaintext1 = "OutOfService"
	Plaintext2 = "XFINITY"
	Plaintext3 = "xfinitywifi"
)

var (
	Encrypted1a, Encrypted1b, Encrypted2a, Encrypted2b, Encrypted3a, Encrypted3b string
)

func NewTestCodec(conf *configuration.Config) *AesCodec {
	var err error
	if testCodec == nil {
		randomKey := GetRandomEncryptionKey()
		testCodec, err = NewAesCodec(conf, randomKey)
		if err != nil {
			panic(err)
		}
	}

	Encrypted1a, err = testCodec.Encrypt(Plaintext1)
	if err != nil {
		panic(err)
	}
	Encrypted1b, err = testCodec.Encrypt(Plaintext1)
	if err != nil {
		panic(err)
	}

	Encrypted2a, err = testCodec.Encrypt(Plaintext2)
	if err != nil {
		panic(err)
	}
	Encrypted2b, err = testCodec.Encrypt(Plaintext2)
	if err != nil {
		panic(err)
	}

	Encrypted3a, err = testCodec.Encrypt(Plaintext3)
	if err != nil {
		panic(err)
	}
	Encrypted3b, err = testCodec.Encrypt(Plaintext3)
	if err != nil {
		panic(err)
	}

	return testCodec
}
