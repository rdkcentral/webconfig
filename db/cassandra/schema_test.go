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
package cassandra

import (
	"testing"

	"gotest.tools/assert"
)

func TestTableTimestampColumns(t *testing.T) {
	exp1 := map[string]string{
		"optimization_sns_sent_time":    "optimization_sns_sent_time",
		"optimization_task_failed_time": "optimization_task_failed_time",
		"optimization_time":             "optimization_time",
	}
	res1, err := tdbclient.GetTsColumns("mesh_agent")
	assert.NilError(t, err)
	d := map[string]string{}
	for _, r := range res1 {
		d[r] = r
	}
	assert.DeepEqual(t, exp1, d)

	exp2 := []string{
		"updated_datetime",
	}
	res2, err := tdbclient.GetTsColumns("fingerprint_agent")
	assert.NilError(t, err)
	assert.DeepEqual(t, exp2, res2)

	exp3 := []string{
		"update_time",
	}
	res3, err := tdbclient.GetTsColumns("cujo_security_agent")
	assert.NilError(t, err)
	assert.DeepEqual(t, exp3, res3)
}
