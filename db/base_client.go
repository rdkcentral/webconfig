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
	"github.com/rdkcentral/webconfig/common"
)

type BaseClient struct {
}

// ==== TODO new functions to be implemented by all implementation ====
func (c *BaseClient) SetDocument(cpeMac string, doc *common.Document) error {
	return nil
}

// ==== TODO should be removed later ====
func (c *BaseClient) FactoryReset(cpeMac string) error {
	return nil
}

func (c *BaseClient) FirmwareUpdate(cpeMac string, oldBitmap int, rootDoc *common.RootDocument) error {
	return nil
}

func (c *BaseClient) AppendProfiles(cpeMac string, inbytes []byte) ([]byte, error) {
	return inbytes, nil
}
