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

import (
	"crypto/rand"
	"encoding/base64"
	"os"

	"github.com/go-akka/configuration"
	"github.com/rdkcentral/webconfig/common"
)

var (
	tcodec *AesCodec
)

func GetTestCodec(conf *configuration.Config) (*AesCodec, error) {
	if tcodec == nil {
		envName := conf.GetString("webconfig.security.encryption_key_env_name", envNameDefault)
		if ss := os.Getenv(envName); len(ss) == 0 {
			os.Setenv(envName, GenerateRandomKey())
		}

		codec, err := NewAesCodec(conf)
		if err != nil {
			return nil, common.NewError(err)
		}
		tcodec = codec
	}
	return tcodec, nil
}

// in base64 format
func GenerateRandomKey() string {
	// use 16-byte for simplicity
	bbytes := make([]byte, 16)
	rand.Read(bbytes)
	return base64.StdEncoding.EncodeToString(bbytes)
}
