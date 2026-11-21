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
	"os"
	"path/filepath"
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

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "webconfig.conf")
	assert.NilError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestNewServerConfigHoconValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "root assignments", content: "a = 1\nb = two\n"},
		{name: "nested objects", content: "a { b { c = 1 } }\n"},
		{name: "arrays", content: "a = [1, 2, 3]\n"},
		{name: "nested arrays and objects", content: "a = [[1, 2], {x = 1}]\n"},
		{name: "quoted keys and values", content: "\"quoted key\" = \"quoted value\"\nunquoted = value\n"},
		{name: "colon separator", content: "a: 1\nb : { c = 2 }\n"},
		{name: "omitted separator before object", content: "a { b { c = 1 } }\n"},
		{name: "optional and trailing commas", content: "a = [1, 2,]\nb { x = 1, y = 2, }\n"},
		{name: "hash and double-slash comments", content: "# hash comment\na = 1 // slash comment\nb = 2 # {\n"},
		{name: "braces and brackets in strings", content: "a = \"{[}]\"\n"},
		{name: "comment characters in strings", content: "a = \"# not // comment\"\n"},
		{name: "escaped quotes and backslashes", content: "a = \"say \\\"hi\\\" \\\\ path\"\n"},
		{name: "triple-quoted multiline", content: "a = \"\"\"line1\nline2 { [\n\"\"\"\n"},
		{name: "substitution", content: "a { b = 1 }\nc = ${a.b}\n"},
		{name: "optional substitution", content: "c = ${?WEBCONFIG_UNSET_VAR_FOR_TEST}\n"},
		{name: "substitution with surrounding text", content: "a { b = 1 }\nc = prefix-${a.b}-suffix\n"},
		{name: "duplicate keys and object merge", content: "a { x = 1 }\na { y = 2 }\na.z = 3\n"},
		{name: "plus-equals append", content: "a = [1]\na += 2\n"},
		{name: "dotted keys", content: "metadata.retry_max = 10\nconsumer.session_timeout_sec = 30\n"},
		{name: "unquoted absolute path value", content: "public_key_file = /etc/xpc/webconfig_key_pub.pem\n"},
		{name: "duration and byte-size values", content: "t = 30 seconds\nb = 512k\n"},
		{name: "missing closing curly brace", content: "webconfig { log { level = info }\n", wantErr: "invalid config file"},
		{name: "extra closing curly brace", content: "a { x = 1 }}\nb = two\n", wantErr: "invalid config file"},
		{name: "missing closing square bracket", content: "a = [1, 2\n", wantErr: "invalid config file"},
		{name: "extra closing square bracket", content: "a = [1, 2]]\n", wantErr: "invalid config file"},
		{name: "unterminated quoted string", content: "a = \"abc\nb = 1\n", wantErr: "invalid config file"},
		{name: "unterminated triple-quoted string", content: "a = \"\"\"abc\n", wantErr: "invalid config file"},
		{name: "malformed substitution open", content: "a = ${\n", wantErr: "invalid config file"},
		{name: "malformed substitution empty", content: "a = ${}\n", wantErr: "invalid config file"},
		{name: "whitespace-concatenated array element", content: "a = [1 2]\n"},
		{name: "malformed array adjacent commas", content: "a = [1,, 2]\n", wantErr: "invalid config file"},
		{name: "malformed nested object", content: "a { b { c = 1 }\n", wantErr: "invalid config file"},
		{name: "closing brace as value", content: "x = }\n", wantErr: "invalid config file"},
		{name: "missing assignment separator", content: "x 1\n", wantErr: "invalid config file"},
		{name: "string running past comment", content: "a = \"x # y\nb = 1\n", wantErr: "invalid config file"},
		{name: "unresolved substitution reaches go-akka panic", content: "c = ${a.missing}\n", wantErr: "hocon parse error"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sc, err := NewServerConfig(writeTempConfig(t, tc.content))
			if tc.wantErr == "" {
				assert.NilError(t, err)
				assert.Assert(t, sc != nil)
				return
			}
			assert.ErrorContains(t, err, tc.wantErr)
			assert.Assert(t, sc == nil)
		})
	}
}
