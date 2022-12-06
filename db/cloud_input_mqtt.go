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
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
)

// (1) need to handle if c.IsDbNotFound(err) {
// (2) need to handle root doc version, my guess is another subdoc as "root"
// (3) args is for s.BlockedSubdocIds()
func BuildMqttSendDocument(c DatabaseClient, cpeMac string, fields log.Fields) (*common.Document, error) {
	document, err := c.GetDocument(cpeMac)
	if err != nil {
		// 404 should be included here
		return nil, common.NewError(err)
	}

	filteredDocument := document.FilterForMqttSend()
	if filteredDocument.Length() == 0 {
		return filteredDocument, nil
	}
	for _, subdocId := range c.BlockedSubdocIds() {
		filteredDocument.DeleteSubDocument(subdocId)
	}

	rootDocument, err := c.GetRootDocument(cpeMac)
	if err != nil {
		return nil, common.NewError(err)
	}
	filteredDocument.SetRootDocument(rootDocument)

	return filteredDocument, nil
}

// NOTE the versionMap should be a filtered one, so no extra checking is needed
func UpdateStatesInBatch(c DatabaseClient, cpeMac, metricsAgent string, fields log.Fields, oldSubdocStateMap map[string]int) error {
	for subdocId, oldState := range oldSubdocStateMap {
		newState := common.InDeployment
		updatedTime := int(time.Now().UnixNano() / 1000000)
		subdoc := common.NewSubDocument(nil, nil, &newState, &updatedTime, nil, nil)
		err := c.SetSubDocument(cpeMac, subdocId, subdoc, fields, oldState, metricsAgent)
		if err != nil {
			return common.NewError(err)
		}
	}
	return nil
}
