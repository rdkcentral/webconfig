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

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestMetrics(t *testing.T) {
	sc, err := GetTestServerConfig()
	assert.NilError(t, err)
	m := NewMetrics(sc.Config)
	oldState, newState := 4, 1
	labels := prometheus.Labels{}
	cpeMac := "777700001111"
	fields := log.Fields{}
	m.UpdateStateMetrics(oldState, newState, labels, cpeMac, fields)
}
