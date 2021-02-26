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
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/rdkcentral/webconfig/common"
	"os"
)

func GenerateKeyPairAsFiles(privateKeyFileName string, publicKeyFileName string) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, common.NewError(err)
	}
	privateBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateBlock := &pem.Block{
		Type:  "RSA_PRIVATE_KEY",
		Bytes: privateBytes,
	}
	privateFile, err := os.Create(privateKeyFileName)
	if err != nil {
		return nil, common.NewError(err)
	}
	if err := pem.Encode(privateFile, privateBlock); err != nil {
		return nil, common.NewError(err)
	}
	if err := privateFile.Close(); err != nil {
		return nil, common.NewError(err)
	}

	publicBytes, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return nil, common.NewError(err)
	}
	publicBlock := &pem.Block{
		Type:  "PUBLIC_KEY",
		Bytes: publicBytes,
	}
	publicFile, err := os.Create(publicKeyFileName)
	if err != nil {
		return nil, common.NewError(err)
	}
	if err := pem.Encode(publicFile, publicBlock); err != nil {
		return nil, common.NewError(err)
	}
	if err := publicFile.Close(); err != nil {
		return nil, common.NewError(err)
	}
	return privateKey, nil
}
