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
	"testing"
)

func TestValidateURL(t *testing.T) {
	client := &HttpClient{allowInternalURLs: false}

	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		// Valid public URLs
		{"valid HTTPS", "https://api.example.com/v1/data", false},
		{"valid HTTP", "http://example.com:8080/path?query=value", false},
		{"valid public IP", "http://8.8.8.8/api", false},

		// Empty/malformed URLs
		{"empty URL", "", true},
		{"invalid format", "ht!tp://invalid", true},
		{"missing scheme", "example.com/path", true},

		// Protocol validation
		{"file protocol", "file:///etc/passwd", true},
		{"ftp protocol", "ftp://internal.server/data", true},
		{"ssh protocol", "ssh://server.com", true},

		// Localhost rejection
		{"localhost by name", "http://localhost/api", true},
		{"localhost uppercase", "http://LOCALHOST/api", true},
		{"loopback IPv4", "http://127.0.0.1/admin", true},
		{"loopback IPv4 alternate", "http://127.0.0.2/internal", true},
		{"loopback IPv6", "http://[::1]/internal", true},

		// Private IP ranges (RFC1918)
		{"private 10.x.x.x", "http://10.0.0.1/api", true},
		{"private 10.x.x.x high", "http://10.255.255.254/api", true},
		{"private 172.16.x.x", "http://172.16.5.10/api", true},
		{"private 172.31.x.x", "http://172.31.255.254/api", true},
		{"private 192.168.x.x", "http://192.168.1.1/api", true},
		{"private 192.168.x.x high", "http://192.168.255.254/api", true},

		// Link-local addresses
		{"link-local 169.254.x.x", "http://169.254.169.254/metadata", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateURL(tt.url)
			if (err != nil) != tt.wantError {
				t.Errorf("validateURL(%q) error = %v, wantError %v", tt.url, err, tt.wantError)
			}
		})
	}
}

func TestValidateURLWithInternalAllowed(t *testing.T) {
	client := &HttpClient{allowInternalURLs: true}

	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		// With allowInternalURLs=true, private IPs should be allowed
		{"private IP allowed", "http://10.0.0.1/api", false},
		{"localhost allowed", "http://localhost/api", false},
		{"loopback allowed", "http://127.0.0.1/admin", false},

		// But invalid protocols and malformed URLs should still be rejected
		{"file protocol still rejected", "file:///etc/passwd", true},
		{"empty URL still rejected", "", true},
		{"invalid format still rejected", "ht!tp://invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateURL(tt.url)
			if (err != nil) != tt.wantError {
				t.Errorf("validateURL(%q) with allowInternalURLs=true error = %v, wantError %v", tt.url, err, tt.wantError)
			}
		})
	}
}

func TestValidateURLEdgeCases(t *testing.T) {
	client := &HttpClient{allowInternalURLs: false}

	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		// Edge cases
		{"HTTPS with port", "https://example.com:443/path", false},
		{"HTTP with custom port", "http://example.com:9000/path", false},
		{"URL with query params", "https://api.example.com/v1/data?key=value&foo=bar", false},
		{"URL with fragment", "https://example.com/page#section", false},
		{"URL with auth (should work)", "https://user:pass@example.com/api", false},

		// IPv6 addresses
		{"public IPv6", "http://[2001:4860:4860::8888]/api", false},
		{"IPv6 loopback", "http://[::1]/internal", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateURL(tt.url)
			if (err != nil) != tt.wantError {
				t.Errorf("validateURL(%q) error = %v, wantError %v", tt.url, err, tt.wantError)
			}
		})
	}
}
