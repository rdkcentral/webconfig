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
package sqlite

import (
	"testing"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

// TestUpdateSubDocumentResetsErrorFields verifies that transitioning a subdocument
// to InDeployment (state 3) via UpdateSubDocument clears any stale error_code and
// error_details left from a prior Failure (state 4).
func TestUpdateSubDocumentResetsErrorFields(t *testing.T) {
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "privatessid"

	// step 1: seed a root document so GetRootDocumentLabels succeeds
	rootdoc := &common.RootDocument{}
	err := tdbclient.SetRootDocument(cpeMac, rootdoc)
	assert.NilError(t, err)

	// step 2: write a subdoc in Failure state with non-zero error fields
	srcBytes := common.RandomBytes(100, 150)
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	srcState := common.Failure
	errCode := 204
	errDetails := "failed_retrying:Error unsupported namespace"
	failureSubdoc := common.NewSubDocument(srcBytes, &srcVersion, &srcState, &srcUpdatedTime, &errCode, &errDetails)
	fields := log.Fields{}
	err = tdbclient.SetSubDocument(cpeMac, groupId, failureSubdoc, fields)
	assert.NilError(t, err)

	// verify failure state and error fields persisted
	fetched, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *fetched.State(), common.Failure)
	assert.Equal(t, *fetched.ErrorCode(), 204)
	assert.Equal(t, *fetched.ErrorDetails(), "failed_retrying:Error unsupported namespace")

	// step 3: call UpdateSubDocument (simulating upstream fetch, 2→3 transition)
	// newSubdoc represents fresh config from upstream — no state/error fields set
	newBytes := common.RandomBytes(100, 150)
	newVersion := util.GetMurmur3Hash(newBytes)
	newSubdoc := common.NewSubDocument(newBytes, &newVersion, nil, nil, nil, nil)

	// empty versionMap so UpdateSubDocument does not skip via early-return path
	versionMap := make(map[string]string)
	err = db.UpdateSubDocument(tdbclient, cpeMac, groupId, newSubdoc, failureSubdoc, versionMap, fields)
	assert.NilError(t, err)

	// step 4: verify state advanced to InDeployment and error fields are reset
	fetched, err = tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *fetched.State(), common.InDeployment)
	assert.Equal(t, *fetched.ErrorCode(), 0)
	assert.Equal(t, *fetched.ErrorDetails(), "")
}
