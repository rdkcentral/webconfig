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
package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-akka/configuration"
	"gotest.tools/assert"
)

func TestHttpClientUserAgent(t *testing.T) {
	tests := []struct {
		name     string
		conf     string
		expected string
	}{
		{
			name:     "defaults to webconfig app name",
			conf:     ``,
			expected: "webconfig",
		},
		{
			name:     "uses configured app name",
			conf:     `webconfig.app_name = "myapp"`,
			expected: "myapp",
		},
		{
			name: "explicit http_client user_agent overrides app name",
			conf: `webconfig.app_name = "myapp"
				webconfig.http_client.user_agent = "custom-agent"`,
			expected: "custom-agent",
		},
		{
			name: "empty http_client user_agent falls back to app name",
			conf: `webconfig.app_name = "myapp"
				webconfig.http_client.user_agent = ""`,
			expected: "myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserAgent string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUserAgent = r.UserAgent()
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			conf := configuration.ParseString(tt.conf)
			client := NewHttpClient(conf, "xconf", nil)
			_, _, _, err := client.Do("GET", server.URL, nil, nil, nil, "", 0)
			assert.NilError(t, err)
			assert.Equal(t, tt.expected, gotUserAgent)
		})
	}
}
