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
	"github.com/rdkcentral/webconfig/util"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestStateMetrics(t *testing.T) {
	tmetrics.ResetStateGauges()
	cpeMac := util.GenerateRandomCpeMac()
	groupId := "privatessid"

	// verify starting empty
	fields := log.Fields{}
	_, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.Assert(t, tdbclient.IsDbNotFound(err))

	labels := prometheus.Labels{
		"model":     "unknown",
		"fwversion": "unknown",
		"client":    "default",
	}

	// ==== insert a doc ====
	srcBytes := []byte("hello world")
	srcVersion := util.GetMurmur3Hash(srcBytes)
	srcUpdatedTime := int(time.Now().UnixNano() / 1000000)
	state1 := common.PendingDownload
	sourceDoc := common.NewSubDocument(srcBytes, &srcVersion, &state1, &srcUpdatedTime, nil, nil)
	err = tdbclient.SetSubDocument(cpeMac, groupId, sourceDoc, fields)
	assert.NilError(t, err)

	// verify state metrics
	labels["feature"] = groupId
	scntr, err := tmetrics.GetStateCounter(labels)
	assert.NilError(t, err)
	assert.Equal(t, scntr.PendingDownload, 1)
	assert.Equal(t, scntr.InDeployment, 0)
	assert.Equal(t, scntr.Deployed, 0)
	assert.Equal(t, scntr.Failure, 0)

	// read a SubDocument from db and verify identical
	doc1, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	err = sourceDoc.Equals(doc1)
	assert.NilError(t, err)

	// ==== update an doc with the same cpeMac and a changed state ====
	state2 := common.InDeployment
	doc1.SetState(&state2)
	err = tdbclient.SetSubDocument(cpeMac, groupId, doc1, state1, labels, fields)
	assert.NilError(t, err)

	// verify state metrics
	scntr, err = tmetrics.GetStateCounter(labels)
	assert.NilError(t, err)
	assert.Equal(t, scntr.PendingDownload, 0)
	assert.Equal(t, scntr.InDeployment, 1)
	assert.Equal(t, scntr.Deployed, 0)
	assert.Equal(t, scntr.Failure, 0)

	doc2, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *doc2.State(), common.InDeployment)

	// ==== update an doc with the same cpeMac and a changed state ====
	state3 := common.Deployed
	doc2.SetState(&state3)
	err = tdbclient.SetSubDocument(cpeMac, groupId, doc2, state2, labels, fields)
	assert.NilError(t, err)

	// verify state metrics
	scntr, err = tmetrics.GetStateCounter(labels)
	assert.NilError(t, err)
	assert.Equal(t, scntr.PendingDownload, 0)
	assert.Equal(t, scntr.InDeployment, 0)
	assert.Equal(t, scntr.Deployed, 1)
	assert.Equal(t, scntr.Failure, 0)

	doc3, err := tdbclient.GetSubDocument(cpeMac, groupId)
	assert.NilError(t, err)
	assert.Equal(t, *doc3.State(), common.Deployed)
}
// add a dummy change
