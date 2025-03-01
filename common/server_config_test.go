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
package common

import (
	"testing"

	"gotest.tools/assert"
)

func TestGetTestServerConfig(t *testing.T) {
	sc, err := GetTestServerConfig("../config/sample_webconfig.conf")
	assert.NilError(t, err)

	xclustersNodeValue := sc.GetNode("webconfig.kafka.xclusters")
	assert.Assert(t, xclustersNodeValue == nil)

	ckeys := sc.KafkaClusterNames()
	expectedClusterKeys := []string{"mesh", "east"}
	assert.DeepEqual(t, ckeys, expectedClusterKeys)

	expectedClusterBrokers := []string{
		"localhost:19093",
		"localhost:19094",
	}

	for i, ckey := range ckeys {
		brokers := sc.GetString("webconfig.kafka.clusters." + ckey + ".brokers")
		assert.Equal(t, brokers, expectedClusterBrokers[i])
	}
}

func TestServerConfigAddConfig(t *testing.T) {
	tsc, err := GetTestServerConfig("../config/sample_webconfig.conf")
	assert.NilError(t, err)

	b1 := tsc.GetBoolean("webconfig.upstream_profiles_enabled")
	assert.Assert(t, !b1)
	b2 := tsc.GetBoolean("webconfig.supplementary_appending_enabled")
	assert.Assert(t, b2)
	s1 := tsc.GetString("webconfig.database.active_driver")
	assert.Equal(t, s1, "cassandra")

	items := []string{
		"webconfig.upstream_profiles_enabled = true",
		"webconfig.supplementary_appending_enabled = false",
	}

	tsc.AddConfig(items...)
	b1 = tsc.GetBoolean("webconfig.upstream_profiles_enabled")
	assert.Assert(t, b1)
	b2 = tsc.GetBoolean("webconfig.supplementary_appending_enabled")
	assert.Assert(t, !b2)
	s1 = tsc.GetString("webconfig.database.active_driver")
	assert.Equal(t, s1, "cassandra")
}
